package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/api/types/mount"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CheckpointBody struct {
	LeaveRun      bool     `json:"leave_running"`
	ImgUrl        string   `json:"image_url"`
	Passphrase    string   `json:"passphrase_file"`
	Preserve_path string   `json:"preserved_paths"`
	Num_shards    int      `json:"num_shards"`
	Cpu_budget    string   `json:"cpu_budget"`
	Verbose       int      `json:"verbose"`
	Envs          []string `json:"envs"`
}

type StartBody struct {
	ContainerName string        `json:"container_name"`
	Image         string        `json:"image"`
	AppPorts      []string      `json:"app_ports"`
	Envs          []string      `json:"envs"`
	Mounts        []mount.Mount `json:"mounts"`
	Caps          []string      `json:"caps"`
}

func upHandler(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"message": "up"})
}

func runHandler(c *gin.Context) {
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	containerName := c.Param("name")
	ffRet, ffMsg := callFastFreeze(0, requestBody, containerName)
	if ffRet == 1 {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": ffMsg})
	} else {
		updateServiceStatus(containerName, "running")
		c.IndentedJSON(http.StatusOK, gin.H{"message": ffMsg})
	}
}

func checkpointHandler(c *gin.Context) {
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	containerName := c.Param("name")
	ffRet, ffMsg := callFastFreeze(1, requestBody, containerName)

	if ffRet == 1 {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": ffMsg})
	} else {
		var checkpointBody CheckpointBody
		_ = json.Unmarshal(requestBody, &checkpointBody)
		if checkpointBody.LeaveRun {
			updateServiceStatus(containerName, "running")
		} else {
			updateServiceStatus(containerName, "checkpointed")
		}
		c.IndentedJSON(http.StatusOK, gin.H{"message": ffMsg})
	}
}

func subscribeHandler(c *gin.Context) {
	containerName := c.Query("container_name")
	containerId := c.Query("container_id")
	image := c.Query("image")
	daemonPort := c.Query("daemon_port")

	_, ok := services[containerName]
	if ok {
		msg := "Container with the name " + containerName + " already existed!"
		c.IndentedJSON(http.StatusConflict, gin.H{"message": msg})
		return
	}
	serviceSubscribe(containerName, containerId, image, daemonPort)
	msg := "Container with the name " + containerName + " subscribed"
	c.IndentedJSON(http.StatusOK, gin.H{"message": msg})
}

func unsubscribeHandler(c *gin.Context) {
	containerName := c.Param("name")
	if err := serviceUnsubscribe(containerName); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Error:" + err.Error()})
		return
	}
	msg := "Container with the name " + containerName + " unsubscribed"
	c.IndentedJSON(http.StatusOK, gin.H{"message": msg})
}

func startHandler(c *gin.Context) {
	var newStart StartBody
	if err := c.BindJSON(&newStart); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	createServiceDir(newStart.ContainerName)
	if err := startService(newStart.ContainerName, newStart.Image, newStart.AppPorts, newStart.Envs, newStart.Mounts, newStart.Caps); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to start the container:" + err.Error()})
		return
	}
	msg := "Container with the name " + newStart.ContainerName + " start successfully"
	c.IndentedJSON(http.StatusOK, gin.H{"message": msg})

}

func stopHandler(c *gin.Context) {
	containerName := c.Param("name")
	if err := stopContainer(containerName); err != nil {
		fmt.Printf("Stop container error: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop the container"})
		return
	}
	msg := "Container with the name " + containerName + " stopped successfully"
	c.IndentedJSON(http.StatusOK, gin.H{"message": msg})
}

func removeHandler(c *gin.Context) {
	containerName := c.Param("name")
	if err := removeContainer(containerName); err != nil {
		fmt.Printf("Delete container error: %v", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the container"})
		return
	}
	msg := "Container with the name " + containerName + " deleted successfully"
	c.IndentedJSON(http.StatusOK, gin.H{"message": msg})
}

func getContainerInfoHandler(c *gin.Context) {
	containerName := c.Param("name")
	containerInfo, err := getContainerInfo(services[containerName].ContainerId)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Client cannot get container info!"})
		return
	}
	c.IndentedJSON(http.StatusOK, containerInfo)
}

func getServiceInfoHandler(c *gin.Context) {
	containerName := c.Param("name")
	if _, ok := services[containerName]; ok {
		services[containerName].getUpdateServiceStatus()
		c.IndentedJSON(http.StatusOK, services[containerName])
	} else {
		msg := "no service name " + containerName + " found!"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": msg})
	}

}

func getAllServicesInfoHandler(c *gin.Context) {
	allServices := make([]Service, len(services))
	i := 0
	for _, service := range services {
		stat := service.getUpdateServiceStatus()
		if stat == "" {
			continue
		}
		allServices[i] = service
		i++
	}
	c.IndentedJSON(http.StatusOK, allServices)
}

func callFastFreeze(mode int, requestBody []byte, containerName string) (int, string) {
	var daemonPort string
	service, ok := services[containerName]
	if ok {
		daemonPort = service.DaemonPort
	} else {
		return 1, "Container not in the team, Try Subscribe or Start it first"
	}
	url := "http://127.0.0.1:" + daemonPort
	if mode == 0 {
		url += "/run"
	} else {
		url += "/checkpoint"
	}
	fmt.Println(url)
	// Create an HTTP Post request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Error("Error creating the request", zap.Error(err))
		return 1, "Error creating the request"
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	logger.Debug("Going to send the request", zap.String("url", url), zap.ByteString("requestBody", requestBody))

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error sending the request", zap.Error(err))
		return 1, "Error sending the request"
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading the response", zap.Error(err))
		return 1, "Error reading the response"
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("Error response from ff_daemon", zap.Int("statusCode", resp.StatusCode), zap.String("body", string(body)))
		return 1, string(body)
	}

	return 0, string(body)
}
