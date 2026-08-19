package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

const mergeBlockWait = 250 * time.Millisecond

type taggedEvent struct {
	Session string `json:"session"`
	agentTypes.Event
}

type connectedFrame struct {
	Session string `json:"session,omitempty"`
	agentTypes.Event
	State   string `json:"state,omitempty"`
	EndedAt string `json:"ended_at,omitempty"`
}

func newConnectedFrame(sessionID string) connectedFrame {
	status := configStatus.Get(sessionID)
	return connectedFrame{
		Session: sessionID,
		Event:   agentTypes.Event{Type: agentTypes.EventConnected, Text: sessionID},
		State:   status.State,
		EndedAt: status.EndedAt,
	}
}

func StreamMultiLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.Query("sessions"))
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessions query param is required"})
			return
		}

		var sids []string
		for s := range strings.SplitSeq(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sids = append(sids, s)
			}
		}
		if len(sids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid session ids"})
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		var subs []*pubsub.Subscriber

		replay := c.DefaultQuery("replay", "1") != "0"

		for _, sid := range sids {
			if raw, err := json.Marshal(newConnectedFrame(sid)); err == nil {
				fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
			}

			if !replay {
				continue
			}

			for _, ev := range sessionLog.RecentEvents(sid, 512) {
				if toolRegister.IsSystemUse(ev.ToolName) {
					continue
				}
				te := taggedEvent{Session: sid, Event: ev}
				if raw, err := json.Marshal(te); err == nil {
					fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
				}
			}
		}
		c.Writer.Flush()

		merged := make(chan taggedEvent, 1024)
		var fanInDropped atomic.Int64
		for _, sid := range sids {
			sub := pubsub.Sub(sid, 1024)
			subs = append(subs, sub)

			go func(id string, s *pubsub.Subscriber) {
				for ev := range s.Events() {
					if toolRegister.IsSystemUse(ev.ToolName) {
						continue
					}
					te := taggedEvent{Session: id, Event: ev}
					select {
					case merged <- te:
						continue
					default:
					}

					timer := time.NewTimer(mergeBlockWait)
					select {
					case merged <- te:
					case <-timer.C:
						n := fanInDropped.Add(1)
						slog.Warn("multilog fan-in overflow, event dropped",
							slog.String("session", id),
							slog.String("event", ev.Type.String()),
							slog.Int64("dropped_total", n))
					}
					timer.Stop()
				}
			}(sid, sub)
		}

		defer func() {
			for _, s := range subs {
				s.Close()
			}
		}()

		ctx := c.Request.Context()
		heartbeat := time.NewTicker(logHeartbeat)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case te, ok := <-merged:
				if !ok {
					return
				}
				raw, err := json.Marshal(te)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
					return
				}
				c.Writer.Flush()
				if te.Type == agentTypes.EventDone {
					dropped := fanInDropped.Swap(0)
					for _, s := range subs {
						dropped += s.TakeDropped()
					}
					if dropped > 0 {
						if _, err := fmt.Fprintf(c.Writer, "data: {\"session\":%q,\"type\":\"EventTruncated\",\"dropped\":%d}\n\n", te.Session, dropped); err != nil {
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
