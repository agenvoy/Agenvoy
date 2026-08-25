package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func GetVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": runtime.CurrentVersion,
			"dev":     runtime.IsDev(),
		})
	}
}
