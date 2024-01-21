package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var workerId string
var managerAddr string

func main() {
	logger := getGlobalLogger()
	err := ctrl_args()
	if err != nil {
		logger.Error("Error parsing arguments", zap.Error(err))
		return
	}
	createRootServiceDir()
	checkServices()
	r := gin.Default()
	// define the routes
	r.GET("/cm_controller/v1/up", upHandler)
	r.POST("/cm_controller/v1/run/:name", runHandler)
	r.POST("/cm_controller/v1/checkpoint/:name", checkpointHandler)
	r.POST("/cm_controller/v1/subscribe", subscribeHandler)
	r.POST("/cm_controller/v1/unsubscribe/:name", unsubscribeHandler)
	r.POST("/cm_controller/v1/start", startHandler)
	r.POST("/cm_controller/v1/stop/:name", stopHandler)
	r.DELETE("/cm_controller/v1/remove/:name", removeHandler)
	r.GET("/cm_controller/v1/service/container_info/:name", getContainerInfoHandler)
	r.GET("/cm_controller/v1/service/:name", getServiceInfoHandler)
	r.GET("/cm_controller/v1/service", getAllServicesInfoHandler)

	go func() {
		err := r.Run(":8787")
		if err != nil {
			logger.Error("impossible to start server", zap.Error(err))
		}
		logger.Info("server started")
	}()
	//createRootServiceDir()

	go func() {
		for {
			logger.Debug("Sending heartbeat")
			sendHeartbeat()
			time.Sleep(3 * time.Second) // Heartbeat every 3 seconds (adjust as needed)
		}
	}()

	select {}
}

func ctrl_args() error {
	args := os.Args[1:]
	if len(args) != 4 {
		fmt.Println("Usage: ./cm_controller --worker,-w <worker id> --manager,-m <manager address>")
		return errors.New("Invalid number of arguments")
	}
	c := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--worker" || args[i] == "-w" {
			workerId = args[i+1]
			c++
		}
		if args[i] == "--manager" || args[i] == "-m" {
			managerAddr = args[i+1]
			c++
		}
	}
	if c == 2 {
		return nil
	} else {
		fmt.Println("Usage: ./cm_controller --worker,-w <worker id> --manager,-m <manager address>")
		return errors.New("Invalid arguments")
	}
}
