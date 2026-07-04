package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func ListModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models := []string{configBot.DefaultModel}
		if cfg, err := config.Load(); err == nil && cfg != nil {
			for _, m := range cfg.Models {
				if name := strings.TrimSpace(m.Name); name != "" {
					models = append(models, name)
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}

func SetSessionModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sid)) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		var body struct {
			Model string `json:"model"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		model := strings.TrimSpace(body.Model)
		if model == "" {
			model = configBot.DefaultModel
		}
		if model != configBot.DefaultModel {
			valid := false
			if cfg, err := config.Load(); err == nil && cfg != nil {
				for _, m := range cfg.Models {
					if m.Name == model {
						valid = true
						break
					}
				}
			}
			if !valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown model: " + model})
				return
			}
		}

		configBot.SetModel(sid, model, "")
		c.JSON(http.StatusOK, gin.H{"ok": true, "model": model})
	}
}
