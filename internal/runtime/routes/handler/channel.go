package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func channelAuthPath(channel string) (string, bool) {
	switch channel {
	case "telegram":
		return filesystem.TelegramAuthPath, true
	case "discord":
		return filesystem.DiscordAuthPath, true
	}
	return "", false
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
