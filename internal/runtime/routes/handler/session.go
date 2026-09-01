package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	provider "github.com/pardnchiu/go-llm-router/core"

	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/agenvoy/internal/session/config"
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

func sessionDetail(sid string) gin.H {
	selfID, name, rule := configBot.GetPersona(sid)
	model, reasoning := configBot.GetModel(sid)
	levels := reasoningLevels()
	if !slices.Contains(levels, reasoning) {
		reasoning = provider.ReasoningDefault.String()
	}
	status := configStatus.Get(sid)
	return gin.H{
		"id":        sid,
		"self_id":   selfID,
		"name":      name,
		"rule":      rule,
		"state":     status.State,
		"model":     model,
		"reasoning": reasoning,
		"levels":    levels,
		"count":     status.Count,
	}
}

func GetSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, ok := sessionParam(c)
		if !ok {
			return
		}
		out := sessionDetail(sid)
		if c.Query("chat") == "1" {
			content, err := sessionChatLog(sid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out["chat"] = content
		}
		if c.Query("usage") == "1" {
			periods, err := sessionUsage(sid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out["usage"] = periods
		}
		c.JSON(http.StatusOK, out)
	}
}

func UpdateSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, ok := sessionParam(c)
		if !ok {
			return
		}

		var body struct {
			SelfID    *string `json:"self_id"`
			Name      *string `json:"name"`
			Rule      *string `json:"rule"`
			Model     *string `json:"model"`
			Reasoning *string `json:"reasoning"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		model := ""
		if body.Model != nil {
			model = strings.TrimSpace(*body.Model)
			if model == "" {
				model = configBot.DefaultModel
			}
			if model != configBot.DefaultModel && !registeredModel(model) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown model: " + model})
				return
			}
		}

		reasoning := ""
		if body.Reasoning != nil {
			reasoning = strings.TrimSpace(*body.Reasoning)
			levels := reasoningLevels()
			if !slices.Contains(levels, reasoning) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("reasoning must be one of %v", levels)})
				return
			}
		}

		if body.SelfID != nil || body.Name != nil || body.Rule != nil {
			selfID, name, rule := configBot.GetPersona(sid)
			if body.SelfID != nil {
				selfID = strings.TrimSpace(*body.SelfID)
			}
			if body.Name != nil {
				name = *body.Name
			}
			if body.Rule != nil {
				rule = *body.Rule
			}
			if err := historyStore.ValidSelfID(selfID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := configBot.SavePersona(sid, selfID, name, rule); err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, historyStore.ErrDuplicateSelfID) {
					status = http.StatusConflict
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}
		}

		if model != "" || reasoning != "" {
			configBot.SetModel(sid, model, reasoning)
		}

		c.JSON(http.StatusOK, sessionDetail(sid))
	}
}

func registeredModel(name string) bool {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return false
	}
	return slices.ContainsFunc(cfg.Models, func(m config.ModelEntry) bool { return m.Name == name })
}

func DeleteSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, ok := sessionParam(c)
		if !ok {
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
