package handler

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func modelNames() []string {
	names := []string{configBot.DefaultModel}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		for _, m := range cfg.Models {
			if name := strings.TrimSpace(m.Name); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func modelObject(name string) gin.H {
	owner := "agenvoy"
	if prefix, _, ok := strings.Cut(name, "@"); ok && prefix != "" {
		owner = prefix
	}
	return gin.H{
		"id":       name,
		"object":   "model",
		"created":  0,
		"owned_by": owner,
	}
}

func ListModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models := modelNames()

		data := make([]gin.H, 0, len(models))
		for _, name := range models {
			data = append(data, modelObject(name))
		}

		c.JSON(http.StatusOK, gin.H{
			"models": models,
			"object": "list",
			"data":   data,
		})
	}
}

func GetModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimPrefix(c.Param("id"), "/")
		if id == "" || !slices.Contains(modelNames(), id) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"message": fmt.Sprintf("The model '%s' does not exist", id),
				"type":    "invalid_request_error",
				"param":   "model",
				"code":    "model_not_found",
			}})
			return
		}
		c.JSON(http.StatusOK, modelObject(id))
	}
}
