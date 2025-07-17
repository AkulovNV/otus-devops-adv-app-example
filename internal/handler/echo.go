package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Echo(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"query": c.Request.URL.Query(),
			"error": "invalid or missing JSON body",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"echo": body,
	})
}
