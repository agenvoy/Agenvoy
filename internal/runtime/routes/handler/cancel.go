package handler

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
)

func CancelSessionTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := strings.TrimSpace(c.Param("session_id"))
		taskID := strings.TrimSpace(c.Param("task_id"))
		if sessionID == "" || taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_id are required"})
			return
		}

		status := configStatus.Get(sessionID)
		if !slices.ContainsFunc(status.Active, func(t configStatus.Task) bool { return t.ID == taskID }) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not active in this session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": exec.CancelTask(taskID)})
	}
}
