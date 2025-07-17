package main

import (
	"github.com/AkulovNV/otus-devops-adv-app-example/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/echo", handler.Echo)
	r.GET("/echo", handler.Echo)

	r.Run(":8080")
}
