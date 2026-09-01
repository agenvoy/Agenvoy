package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
)

func CancelSessionTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := strings.TrimSpace(c.Param("session_id"))
		onceID := strings.TrimSpace(c.Param("once_id"))
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		if onceID != "" && onceID != "current" && exec.CancelTask(onceID) {
			c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": true})
			return
		}

		event := agentTypes.Event{Type: agentTypes.EventCanceled, OnceID: onceID}
		sessionLog.Record(sessionID, event)
		pubsub.Pub(sessionID, event)
		c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": false, "stale": true})
	}
}
