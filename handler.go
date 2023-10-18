package main

import (
  "net/http"
  "io/ioutil"
	"github.com/gin-gonic/gin"
  "bytes"
)

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


func callFastFreeze(mode int, requestBody []byte, container_name string) (int, string) {
  var daemon_port string
  service, ok := services[container_name]
  if ok {
    daemon_port = service.daemon_port
  }else {
    return 1, "Container not in the team, Try Subscribe it first"
  }
	url := "http://localhost:" + daemon_port
  if mode == 0 {
    url += "/run"
  } else {
    url += "/checkpoint"
  }

  // Create an HTTP Post request
  req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
  if err != nil {
    return 1, "Error creating the request"
  }
    // Send the HTTP request
  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
      return 1, "Error sending the request"
  }
  defer resp.Body.Close()

  // Read the response body
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
      return 1, "Error reading the response"
  }

  return 0, string(body)
}