package main

import (
  "net/http"
  "io/ioutil"
	"github.com/gin-gonic/gin"
  "bytes"
  "fmt"
  "github.com/docker/docker/api/types/mount"
)

type Start_body struct {
  Container_name string `json:"container_name"`
  Image string `json:"image"` 
  App_port string `json:"app_port"`
  Envs []string `json:"envs"`
  Mounts []mount.Mount `json:"mounts"`
}

func run_handler(c *gin.Context) {
  requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
	}
  container_name := c.Query("container_name")
  ff_ret, ff_msg := callFastFreeze(0, requestBody, container_name)
  if ff_ret == 1 {
    c.JSON(http.StatusInternalServerError,gin.H{"message": ff_msg})
  }else {
    c.JSON(http.StatusOK, gin.H{"message": ff_msg})
  }
}

func checkpoint_handler(c *gin.Context) {
  requestBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
	}
  container_name := c.Query("container_name")
  ff_ret, ff_msg := callFastFreeze(1, requestBody,container_name)
  if ff_ret == 1 {
    c.JSON(http.StatusInternalServerError,gin.H{"message": ff_msg})
  }else {
    c.JSON(http.StatusOK, gin.H{"message": ff_msg})
  }
}

func subscribe_handler(c *gin.Context){
  container_name := c.Query("container_name")
  image := c.Query("image")
  daemon_port := c.Query("daemon_port")

  _, ok := services[container_name]
  if ok {
    msg := "Container with the name " + container_name + " already existed!"  
    c.JSON(http.StatusBadRequest,gin.H{"message": msg}) 
    return
  }
  container_subscribe(container_name, image, daemon_port)
  msg := "Container with the name " + container_name + " subscribed" 
  c.JSON(http.StatusOK, gin.H{"message": msg}) 
}

func start_handler(c *gin.Context){
  var new_start Start_body
  if err := c.BindJSON(&new_start); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
    return
  }
 
  if err := run_container(new_start.Container_name, new_start.Image, new_start.App_port, new_start.Envs, new_start.Mounts); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run the container"})
    return
  }
  msg := "Container with the name " + new_start.Container_name + " run successfully" 
  c.JSON(http.StatusOK, gin.H{"message": msg}) 

}


func callFastFreeze(mode int, requestBody []byte, container_name string) (int, string) {
  var daemon_port string
  service, ok := services[container_name]
  if ok {
    daemon_port = service.Daemon_port
  }else {
    return 1, "Container not in the team, Try Subscribe or Start it first"
  }
	url := "http://127.0.0.1:" + daemon_port
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
