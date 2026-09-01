package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
)

var usagePeriods = []struct {
	label string
	days  int
}{
	{label: "24h", days: 1},
	{label: "7d", days: 7},
	{label: "28d", days: 28},
}

func sessionChatLog(sid string) (string, error) {
	path := filesystem.ActionLogPath(sid)
	if !go_pkg_filesystem_reader.Exists(path) {
		return "", nil
	}
	return go_pkg_filesystem.ReadText(path)
}

func sessionUsage(sid string) (map[string]map[string]usagelog.ModelUsage, error) {
	now := time.Now()
	periods := make(map[string]map[string]usagelog.ModelUsage, len(usagePeriods))
	for _, period := range usagePeriods {
		summary, err := usagelog.Usage(sid, period.days, now)
		if err != nil {
			return nil, err
		}
		periods[period.label] = summary
	}
	return periods, nil
}

func GetTotalUsage() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()

		periods := make(map[string]map[string]usagelog.ModelUsage, len(usagePeriods))
		for _, period := range usagePeriods {
			summary, err := usagelog.Total(period.days, now)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			periods[period.label] = summary
		}
		c.JSON(http.StatusOK, gin.H{"periods": periods})
	}
}

func ListTaskHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		rows, err := historyStore.ListAction(c.Request.Context(), sid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
		tasks := make([]gin.H, 0, len(rows))
		for _, row := range rows {
			if !actionMatch(row, keyword) {
				continue
			}
			tasks = append(tasks, gin.H{
				"task_hash": row.TaskHash,
				"end_at":    row.EndAt.Format(time.RFC3339),
				"objective": row.Objective,
				"model":     row.Model,
				"reasoning": row.Reasoning,
			})
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

func actionMatch(row historyStore.ActionRecord, keyword string) bool {
	if keyword == "" {
		return true
	}
	for _, field := range []string{row.Objective, row.Reply, string(row.ToolResults)} {
		if strings.Contains(strings.ToLower(field), keyword) {
			return true
		}
	}
	return false
}

func GetTaskHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		hash := strings.TrimSpace(c.Param("task_hash"))
		if sid == "" || hash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_hash are required"})
			return
		}

		row, ok, err := historyStore.ReadAction(c.Request.Context(), sid, hash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		raw, err := json.Marshal(row)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"content": string(raw)})
	}
}
