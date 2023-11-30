package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	createRootServiceDir()
	r := gin.Default()
	// define the routes
	r.POST("/cm_run/:name", runHandler)
	r.POST("/cm_checkpoint/:name", checkpointHandler)
	r.POST("/cm_subscribe", subscribeHandler)
	r.POST("/cm_start", startHandler)
	r.GET("/cm_service/container_info/:name", getContainerInfoHandler)
	r.GET("/cm_service/:name", getServiceInfoHandler)
	r.GET("/cm_service", getAllServicesInfoHandler)
	err := r.Run(":8787")
	if err != nil {
		fmt.Printf("impossible to start server: %s\n", err)
	}
	//createRootServiceDir()
}
