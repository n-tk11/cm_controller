package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)


func main() {
   r := gin.Default()
   // define the routes
   r.POST("/cm_run", run_handler)
   r.POST("/cm_checkpoint", checkpoint_handler)
   r.POST("/cm_subscribe",subscribe_handler)
   r.POST("/cm_start", start_handler)
   err := r.Run(":8787")
   if err != nil {
      fmt.Println("impossible to start server: %s", err)
   }
}

