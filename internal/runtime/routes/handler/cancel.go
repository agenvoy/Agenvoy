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
		taskHash := strings.TrimSpace(c.Param("task_hash"))
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		if taskHash != "" && taskHash != "current" && exec.Cancel(taskHash) {
			c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": true})
			return
		}

		event := agentTypes.Event{Type: agentTypes.EventCanceled, TaskHash: taskHash}
		sessionLog.Record(sessionID, event)
		pubsub.Pub(sessionID, event)
		c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": false, "stale": true})
	}
}
