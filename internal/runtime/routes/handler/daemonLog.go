package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
)

const logHeartbeat = 25 * time.Second

func StreamDaemonLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		sub := pubsub.Sub(configs.DAEMON_LOG_CHANNEL, 1024)
		defer sub.Close()

		heartbeat := time.NewTicker(logHeartbeat)
		defer heartbeat.Stop()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				raw, err := json.Marshal(ev)
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
