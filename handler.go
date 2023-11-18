package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	containerName := c.Query("container_name")
	ffRet, ffMsg := callFastFreeze(0, requestBody, containerName)
	if ffRet == 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"message": ffMsg})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": ffMsg})
	}
}

func checkpointHandler(c *gin.Context) {
	requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}
	containerName := c.Query("container_name")
	ffRet, ffMsg := callFastFreeze(1, requestBody, containerName)
	if ffRet == 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"message": ffMsg})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": ffMsg})
	}
}

func subscribeHandler(c *gin.Context) {
	containerName := c.Query("container_name")
	image := c.Query("image")
	daemonPort := c.Query("daemon_port")

	_, ok := services[containerName]
	if ok {
		msg := "Container with the name " + containerName + " already existed!"
		c.JSON(http.StatusBadRequest, gin.H{"message": msg})
		return
	}
	containerSubscribe(containerName, image, daemonPort)
	msg := "Container with the name " + containerName + " subscribed"
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func startHandler(c *gin.Context) {
	var newStart StartBody
	if err := c.BindJSON(&newStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
		return
	}

	if err := runContainer(newStart.ContainerName, newStart.Image, newStart.AppPort, newStart.Envs, newStart.Mounts, newStart.Caps); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run the container"})
		return
	}
	msg := "Container with the name " + newStart.ContainerName + " run successfully"
	c.JSON(http.StatusOK, gin.H{"message": msg})

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

	req.Header.Set("Content-Type", "application/json")
	fmt.Println("Going to send the request")
	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 1, "Error sending the request"
	}
	fmt.Println("Request sent to ff_daemon")
	defer resp.Body.Close()
	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 1, "Error reading the response"
	}

	return 0, string(body)
}
