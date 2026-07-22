package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	historyStore "github.com/pardnchiu/agenvoy/internal/session/history/store"
)

type SessionInfo struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	State   string              `json:"state"`
	Model   string              `json:"model"`
	Active  []configStatus.Task `json:"active"`
	EndedAt string              `json:"ended_at"`
}

func ListSessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := c.DefaultQuery("filter", "all")

		dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"sessions": []SessionInfo{}})
			return
		}

		list := make([]SessionInfo, 0, len(dirs))
		for _, dir := range dirs {
			sid := dir.Name
			if strings.HasPrefix(sid, ".") || sid == "jarvis" {
				continue
			}

			switch filter {
			case "active":
				status := configStatus.Get(sid)
				if status.State != configStatus.StatusOnline {
					continue
				}
			case "permanent":
				if strings.HasPrefix(sid, "temp-") {
					continue
				}
			case "temporary":
				if !strings.HasPrefix(sid, "temp-") {
					continue
				}
			}

			status := configStatus.Get(sid)
			name, _ := configBot.Get(sid)
			model, _ := configBot.GetModel(sid)

			if status.Active == nil {
				status.Active = []configStatus.Task{}
			}

			list = append(list, SessionInfo{
				ID:      sid,
				Name:    name,
				State:   status.State,
				Model:   model,
				Active:  status.Active,
				EndedAt: status.EndedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"sessions": list})
	}
}

func CreateSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Prefix string `json:"prefix"`
		}
		_ = c.ShouldBindJSON(&body)
		prefix := strings.TrimSpace(body.Prefix)
		if prefix == "" {
			prefix = "cli-"
		}

		sid, err := sessionManager.New(prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"session_id": sid})
	}
}

func UpdateSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			SessionID   string `json:"session_id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sid := strings.TrimSpace(body.SessionID)
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sid)) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if err := configBot.Save(sid, body.Name, body.Description, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func DeleteSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			SessionID string `json:"session_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sid := strings.TrimSpace(body.SessionID)
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sid)) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		if db := torii.DB(torii.DBSessionHist); db != nil {
			if keys := db.Keys(sid + ":*"); len(keys) > 0 {
				db.Del(keys...)
			}
		}
		if err := historyStore.Clear(sid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sessionHistory.ClearMutex(sid)
		exec.ClearSteer(sid)
		if err := os.RemoveAll(filesystem.SessionDir(sid)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func GetSessionPersona() gin.HandlerFunc {
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
		name, body := configBot.Get(sid)
		c.JSON(http.StatusOK, gin.H{"name": name, "body": body})
	}
}

func SetSessionPersona() gin.HandlerFunc {
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
			Name string `json:"name"`
			Body string `json:"body"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := configBot.Save(sid, body.Name, body.Body, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
