package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CheckWorkDir() gin.HandlerFunc {
	return func(c *gin.Context) {
		resolved, err := resolveWorkDir(c.Query("path"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"path": resolved})
	}
}
