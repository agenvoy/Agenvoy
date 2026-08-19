package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

const logHeartbeat = 25 * time.Second

func StreamSessionLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		sub := pubsub.Sub(sid, 1024)
		defer sub.Close()

		if raw, err := json.Marshal(newConnectedFrame(sid)); err == nil {
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
				return
			}
			c.Writer.Flush()
		}

		for _, ev := range sessionLog.RecentEvents(sid, 512) {
			if toolRegister.IsSystemUse(ev.ToolName) {
				continue
			}
			raw, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
				return
			}
		}
		c.Writer.Flush()

		ctx := c.Request.Context()
		heartbeat := time.NewTicker(logHeartbeat)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				if toolRegister.IsSystemUse(ev.ToolName) {
					continue
				}
				raw, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
					return
				}
				c.Writer.Flush()
				if ev.Type == agentTypes.EventDone {
					if n := sub.TakeDropped(); n > 0 {
						if _, err := fmt.Fprintf(c.Writer, "data: {\"type\":\"EventTruncated\",\"dropped\":%d}\n\n", n); err != nil {
							return
						}
						c.Writer.Flush()
					}
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}
