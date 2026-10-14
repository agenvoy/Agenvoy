package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
	provider "github.com/pardnchiu/go-llm-router/core"
)

type Response struct {
	Text      string          `json:"text"`
	Model     string          `json:"model,omitempty"`
	Usage     *provider.Usage `json:"usage,omitempty"`
	SessionID string          `json:"session_id"`
}

func sendResult(c *gin.Context, sessionID string, input string, events <-chan agenttypes.Event) {
	var sb strings.Builder
	var resp Response
	resp.SessionID = sessionID

	utils.EventLog("[HTTP]", agenttypes.Event{
		Type: agenttypes.EventText,
		Text: input,
	}, sessionID, input)

	ctx := c.Request.Context()
	for event := range events {
		if ctx.Err() != nil {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": ctx.Err().Error()})
			return
		}

		utils.EventLog("[HTTP]", event, sessionID, "")

		switch event.Type {
		case agenttypes.EventText:
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(event.Text)
		case agenttypes.EventTextDone:

		case agenttypes.EventDone:
			resp.Model = event.Model
			resp.Usage = event.Usage
		case agenttypes.EventCanceled:
			c.JSON(http.StatusOK, gin.H{"session_id": sessionID, "canceled": true})
			return
		case agenttypes.EventError:
			c.JSON(http.StatusInternalServerError, gin.H{"error": event.Err.Error()})
			return
		}
	}

	resp.Text = sb.String()
	c.JSON(http.StatusOK, resp)
}
