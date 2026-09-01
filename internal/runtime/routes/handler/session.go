package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/compact"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

type SessionInfo struct {
	ID     string `json:"id"`
	SelfID string `json:"self_id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Model  string `json:"model"`
	Count  int    `json:"count"`
}

func ListSessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := c.DefaultQuery("filter", "all")

		dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"sessions": []SessionInfo{}})
			return
		}

		type entry struct {
			info     SessionInfo
			activeAt time.Time
		}

		rows, err := historyStore.ListSessionRows(c.Request.Context())
		if err != nil {
			slog.Warn("historyStore.ListSessionRows",
				slog.String("error", err.Error()))
		}
		states, err := historyStore.ListStateRows(c.Request.Context())
		if err != nil {
			slog.Warn("historyStore.ListStateRows",
				slog.String("error", err.Error()))
		}

		entries := make([]entry, 0, len(dirs))
		for _, dir := range dirs {
			sid := dir.Name
			if strings.HasPrefix(sid, ".") || sid == "jarvis" {
				continue
			}

			status := configStatus.FromRow(states[sid])

			switch filter {
			case "active":
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

			row := rows[sid]
			model := row.Model
			if model == "" {
				model = configBot.DefaultModel
			}

			entries = append(entries, entry{
				info: SessionInfo{
					ID:     sid,
					SelfID: row.SelfID,
					Name:   row.Name,
					State:  status.State,
					Model:  model,
					Count:  status.Count,
				},
				activeAt: lastActiveAt(sid),
			})
		}

		slices.SortStableFunc(entries, func(a, b entry) int {
			return b.activeAt.Compare(a.activeAt)
		})

		list := make([]SessionInfo, 0, len(entries))
		for _, e := range entries {
			list = append(list, e.info)
		}

		c.JSON(http.StatusOK, gin.H{"sessions": list})
	}
}

func lastActiveAt(sessionID string) time.Time {
	if info, err := os.Stat(filesystem.ActionLogPath(sessionID)); err == nil {
		return info.ModTime()
	}
	if info, err := os.Stat(filesystem.SessionDir(sessionID)); err == nil {
		return info.ModTime()
	}
	return time.Time{}
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

		if db := torii.Remote(torii.DBSessionHist); db != nil {
			if keys := db.Keys(c.Request.Context(), sid+":*"); len(keys) > 0 {
				db.Del(c.Request.Context(), keys...)
			}
		}
		if err := historyStore.Clear(sid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := historyStore.DeleteState(c.Request.Context(), sid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := historyStore.DeleteSession(c.Request.Context(), sid); err != nil {
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

func CompactSession() gin.HandlerFunc {
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

		ctx, cancel := memoryCtx()
		defer cancel()

		removed, err := compact.SessionHistory(ctx, sid)
		if err != nil {
			slog.Debug("handler.CompactSession",
				slog.String("session", sid),
				slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed})
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
		selfID, name, body := configBot.GetPersona(sid)
		c.JSON(http.StatusOK, gin.H{"self_id": selfID, "name": name, "body": body})
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
			SelfID string `json:"self_id"`
			Name   string `json:"name"`
			Body   string `json:"body"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		selfID := strings.TrimSpace(body.SelfID)
		if err := historyStore.ValidSelfID(selfID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := configBot.SavePersona(sid, selfID, body.Name, body.Body); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, historyStore.ErrDuplicateSelfID) {
				status = http.StatusConflict
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
