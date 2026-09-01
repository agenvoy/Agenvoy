package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/runtime/startup"
)

func GetStartup() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"enabled": startup.Enabled()})
	}
}

func SetStartup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Enable *bool `json:"enable"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Enable == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "enable is required"})
			return
		}

		var (
			detail string
			err    error
		)
		if *body.Enable {
			detail, err = startup.Enable()
		} else {
			detail, err = startup.Disable()
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "enabled": startup.Enabled(), "detail": detail})
	}
}
