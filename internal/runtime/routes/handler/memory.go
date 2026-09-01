package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func memoryCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		2*time.Duration(filesystem.AgentSendTimeoutSec)*time.Second,
	)
}

func sessionParam(c *gin.Context) (string, bool) {
	sid := strings.TrimSpace(c.Param("session_id"))
	if sid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return "", false
	}
	if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sid)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return "", false
	}
	return sid, true
}

func ResetSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, ok := sessionParam(c)
		if !ok {
			return
		}

		var body struct {
			Mode string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := memoryCtx()
		defer cancel()

		var (
			removed int
			err     error
		)
		switch strings.TrimSpace(body.Mode) {
		case "summary":
			removed, err = exec.ResetSessionWithSummary(ctx, sid)
		case "all":
			removed, err = exec.ResetSessionAll(sid)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be 'summary' or 'all'"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed})
	}
}

func SummarySession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, ok := sessionParam(c)
		if !ok {
			return
		}

		ctx, cancel := memoryCtx()
		defer cancel()

		count, err := exec.ForceSummary(ctx, sid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "count": count})
	}
}

func reasoningLevels() []string {
	out := make([]string, 0, int(provider.ReasoningMax)+1)
	for r := provider.ReasoningNone; r <= provider.ReasoningMax; r++ {
		out = append(out, r.String())
	}
	return out
}
