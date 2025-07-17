package main

import (
	"log"
	"os"

	"github.com/AkulovNV/otus-devops-adv-app-example/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := log.New(os.Stdout, "[echo-server] ", log.LstdFlags)

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/echo", handler.Echo)
	r.GET("/echo", handler.Echo)

	logger.Println("Starting server on :8080")

	if err := r.Run(":8080"); err != nil {
		logger.Fatalf("failed to run server: %v", err)
	}

	// r.Run(":8080")
}
