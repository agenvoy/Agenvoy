package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/runtime/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/telegram"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func GetChannelStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"telegram": gin.H{
				"enabled":   cfg.TelegramEnabled,
				"username":  cfg.TelegramUsername,
				"has_token": strings.TrimSpace(keychain.Get(telegram.Key)) != "",
			},
			"discord": gin.H{
				"enabled":   cfg.DiscordEnabled,
				"username":  cfg.DiscordUsername,
				"has_token": strings.TrimSpace(keychain.Get(discord.Key)) != "",
			},
		})
	}
}

func SetTelegramChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		setChannel(c, telegram.Key, func(cfg *config.Config, enabled bool) {
			cfg.TelegramEnabled = enabled
			if enabled {
				cfg.TelegramUsername = ""
			}
		})
	}
}

func SetDiscordChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		setChannel(c, discord.Key, func(cfg *config.Config, enabled bool) {
			cfg.DiscordEnabled = enabled
			if enabled {
				cfg.DiscordUsername = ""
			}
		})
	}
}

func setChannel(c *gin.Context, key string, apply func(cfg *config.Config, enabled bool)) {
	var body struct {
		Action string `json:"action"`
		Token  string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch strings.TrimSpace(body.Action) {
	case "enable":
		token := strings.TrimSpace(body.Token)
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}
		if err := keychain.Set(key, token); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		apply(cfg, true)
		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "disable":
		if err := keychain.Delete(key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		apply(cfg, false)
		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be enable or disable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
