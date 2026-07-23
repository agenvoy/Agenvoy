package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/kuradb"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

func GetKuradbSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		installed := kuradb.IsInstalled()
		if !installed {
			c.JSON(http.StatusOK, gin.H{
				"installed":       false,
				"install_command": "curl -fsSL " + kuradb.InstallURL + " | bash",
			})
			return
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		version, _ := kuradb.Version()
		running := kuradb.IsRunning()
		endpoint, _ := filesystem.GetKuradbEndpoint()

		healthy := false
		if running {
			healthy = kuradb.ProbeHealth(c.Request.Context()) == nil
		}

		c.JSON(http.StatusOK, gin.H{
			"installed": true,
			"enabled":   cfg.KuradbEnabled,
			"running":   running,
			"healthy":   healthy,
			"version":   version,
			"endpoint":  endpoint,
		})
	}
}

func SetKuradbSetting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Action string `json:"action"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		action := strings.TrimSpace(body.Action)

		if !kuradb.IsInstalled() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kuradb not installed", "install_command": "curl -fsSL " + kuradb.InstallURL + " | bash"})
			return
		}

		switch action {
		case "enable":
			if strings.TrimSpace(keychain.Get("OPENAI_API_KEY")) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "OPENAI_API_KEY required before enabling"})
				return
			}
			if err := setKuradbEnabled(true); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "disable":
			if err := setKuradbEnabled(false); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "start":
			if err := kuradb.Start(c.Request.Context()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "stop":
			if err := kuradb.Stop(c.Request.Context()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "restart":
			if err := kuradb.Restart(c.Request.Context()); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action: " + action})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setKuradbEnabled(enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.KuradbEnabled = enabled
	return config.Save(cfg)
}
