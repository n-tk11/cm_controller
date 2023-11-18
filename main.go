package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	// define the routes
	r.POST("/cm_run", runHandler)
	r.POST("/cm_checkpoint", checkpointHandler)
	r.POST("/cm_subscribe", subscribeHandler)
	r.POST("/cm_start", startHandler)
	err := r.Run(":8787")
	if err != nil {
		fmt.Printf("impossible to start server: %s\n", err)
	}
}
