package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/discord"
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

func channelAuthPath(channel string) (string, bool) {
	switch channel {
	case "telegram":
		return filesystem.TelegramAuthPath, true
	case "discord":
		return filesystem.DiscordAuthPath, true
	}
	return "", false
}

func ListChannelChats() gin.HandlerFunc {
	return func(c *gin.Context) {
		channel := strings.TrimSpace(c.Param("channel"))
		path, ok := channelAuthPath(channel)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel must be telegram or discord"})
			return
		}

		prefix := "tg"
		if channel == "discord" {
			prefix = "dc"
		}
		c.JSON(http.StatusOK, gin.H{"chats": adminChats(prefix, path)})
	}
}

func DeleteChannelChat() gin.HandlerFunc {
	return func(c *gin.Context) {
		channel := strings.TrimSpace(c.Param("channel"))
		path, ok := channelAuthPath(channel)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel must be telegram or discord"})
			return
		}

		var body struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id := strings.TrimSpace(body.ID)
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}

		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no authorized chat yet"})
			return
		}
		text, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		kept := make([]string, 0, 8)
		removed := 0
		for line := range strings.SplitSeq(text, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if utils.ParseChatID(line) == id {
				removed++
				continue
			}
			kept = append(kept, strings.TrimRight(line, "\r"))
		}
		if removed == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "chat not authorized"})
			return
		}

		out := ""
		if len(kept) > 0 {
			out = strings.Join(kept, "\n") + "\n"
		}
		if err := go_pkg_filesystem.WriteFile(path, out, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed})
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
