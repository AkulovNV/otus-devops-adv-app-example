package main

import (
	"echo-server/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/echo", handler.Echo)
	r.GET("/echo", handler.Echo)

	r.Run(":8080") // запускаем на 8080
}
