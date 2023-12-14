package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/api/types/mount"
	"github.com/gin-gonic/gin"
)

type StartBody struct {
	ContainerName string        `json:"container_name"`
	Image         string        `json:"image"`
	AppPort       string        `json:"app_port"`
	Envs          []string      `json:"envs"`
	Mounts        []mount.Mount `json:"mounts"`
	Caps          []string      `json:"caps"`
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
		updateServiceStatus(containerName, "checkpointed")
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
	if err := serviceUnsubscribe(containerName); err != 0 {
		msg := "Container with the name " + containerName + " not found!"
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": msg})
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
	if err := startService(newStart.ContainerName, newStart.Image, newStart.AppPort, newStart.Envs, newStart.Mounts, newStart.Caps); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to run the container"})
		return
	}
	msg := "Container with the name " + newStart.ContainerName + " run successfully"
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
	serviceUnsubscribe(containerName)
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
		service.getUpdateServiceStatus()
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
		return 1, "Error creating the request"
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	fmt.Println("Going to send the request")
	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending the request: %s\n", err)
		return 1, "Error sending the request"
	}
	fmt.Println("Request sent to ff_daemon")
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 1, "Error reading the response"
	}

	if resp.StatusCode != http.StatusOK {
		return 1, string(body)
	}

	return 0, string(body)
}
