package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
)

const daemonLogChannel = "daemon"

func StreamDaemonLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		sub := pubsub.Sub(daemonLogChannel, 256)
		defer sub.Close()

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
				if ev.Type != agentTypes.EventDaemonLog {
					continue
				}
				raw, err := json.Marshal(gin.H{
					"level": ev.Source,
					"text":  ev.Text,
					"time":  time.Now().Format("15:04:05"),
				})
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
					return
				}
				c.Writer.Flush()
			case <-heartbeat.C:
				if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}
