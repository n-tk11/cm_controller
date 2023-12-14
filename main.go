package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	createRootServiceDir()
	r := gin.Default()
	// define the routes
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
	err := r.Run(":8787")
	if err != nil {
		fmt.Printf("impossible to start server: %s\n", err)
	}
	//createRootServiceDir()
}
