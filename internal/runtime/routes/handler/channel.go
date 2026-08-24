package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/line"
	"github.com/pardnchiu/agenvoy/internal/runtime/telegram"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/utils"
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
			"line": gin.H{
				"enabled":  cfg.LineEnabled,
				"username": cfg.LineUsername,
				"has_token": strings.TrimSpace(keychain.Get(line.SecretKey)) != "" &&
					strings.TrimSpace(keychain.Get(line.TokenKey)) != "",
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

func SetLineChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Action string `json:"action"`
			Secret string `json:"secret"`
			Token  string `json:"token"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		enabled := false
		switch strings.TrimSpace(body.Action) {
		case "enable":
			secret := strings.TrimSpace(body.Secret)
			token := strings.TrimSpace(body.Token)
			if secret == "" || token == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "channel secret and access token are required"})
				return
			}
			if err := keychain.Set(line.SecretKey, secret); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := keychain.Set(line.TokenKey, token); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			enabled = true
		case "disable":
			_ = keychain.Delete(line.SecretKey)
			_ = keychain.Delete(line.TokenKey)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "action must be enable or disable"})
			return
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cfg == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "config is empty"})
			return
		}
		cfg.LineEnabled = enabled
		if enabled {
			// * line.New writes the display name back once the webhook starts
			cfg.LineUsername = ""
		}
		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type adminChat struct {
	Value string `json:"value"`
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

func adminChats(prefix, path string) []adminChat {
	entries := utils.ListChats(path)
	list := make([]adminChat, 0, len(entries))
	for _, e := range entries {
		list = append(list, adminChat{
			Value: prefix + "@" + e.ID,
			Type:  prefix,
			ID:    e.ID,
			Name:  e.Name,
		})
	}
	return list
}

func GetAdminChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cfg == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "config is empty"})
			return
		}

		current := strings.TrimSpace(cfg.AdminChannel)
		chats := append(
			adminChats("tg", filesystem.TelegramAuthPath),
			adminChats("dc", filesystem.DiscordAuthPath)...,
		)

		authorized := false
		for _, chat := range chats {
			if chat.Value == current {
				authorized = true
				break
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"admin_channel": current,
			"authorized":    current != "" && authorized,
			"chats":         chats,
		})
	}
}

func SetAdminChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Value *string `json:"value"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value is required; send an empty string to clear"})
			return
		}

		value := strings.TrimSpace(*body.Value)
		if value != "" {
			if _, _, ok := exec.ParseAdminChannel(value); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "value must be tg@<chatID> or dc@<channelID>, or empty to clear"})
				return
			}
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cfg == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "config is empty"})
			return
		}

		cfg.AdminChannel = value
		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "admin_channel": value})
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
