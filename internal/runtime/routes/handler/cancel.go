package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
)

func CancelSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := strings.TrimSpace(c.Param("session_id"))
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		status := configStatus.Get(sessionID)
		if len(status.Active) == 0 {
			c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": []string{}})
			return
		}

		cancelled := make([]string, 0, len(status.Active))
		for _, task := range status.Active {
			if exec.CancelTask(task.ID) {
				cancelled = append(cancelled, task.ID)
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "cancelled": cancelled})
	}
}
