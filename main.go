package main

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	logger := getGlobalLogger()
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
	err := r.Run(":8787")
	if err != nil {
		logger.Error("impossible to start server", zap.Error(err))
	}
	logger.Info("server started")
	//createRootServiceDir()
}
