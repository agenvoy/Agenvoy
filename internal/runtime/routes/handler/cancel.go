package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
)

func CancelSessionTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := strings.TrimSpace(c.Param("session_id"))
		taskID := strings.TrimSpace(c.Param("task_id"))
		if sessionID == "" || taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_id are required"})
			return
		}

		if !exec.CancelTask(taskID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not running in this process"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": true})
	}
}
