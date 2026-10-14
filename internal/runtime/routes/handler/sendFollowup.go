package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/followup"
	configbot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	sessionhistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

func GetSessionFollowup() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, ok := sessionParam(c)
		if !ok {
			return
		}

		_, histories := sessionhistory.Get(sessionID)
		if len(histories) == 0 {
			c.JSON(http.StatusOK, gin.H{"title": "", "suggests": []string{}})
			return
		}

		result := followup.Generate(c.Request.Context(), sessionID, histories, configbot.NeedTitle(sessionID))
		if result.Title != "" {
			if err := configbot.SetTitle(sessionID, result.Title); err != nil {
				slog.Debug("configbot.SetTitle",
					slog.String("session", sessionID),
					slog.String("error", err.Error()))
			}
		}

		suggests := result.Suggests
		if suggests == nil {
			suggests = []string{}
		}
		c.JSON(http.StatusOK, gin.H{"title": result.Title, "suggests": suggests})
	}
}
