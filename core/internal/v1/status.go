package v1

import (
	"github.com/gin-gonic/gin"
)

func getStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}