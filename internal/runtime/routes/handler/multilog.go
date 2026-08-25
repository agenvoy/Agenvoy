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
	internalUtils "github.com/pardnchiu/agenvoy/internal/utils"
)

const (
	mergeBlockWait   = 250 * time.Millisecond
	logHeartbeat     = 25 * time.Second
	daemonLogChannel = "daemon"
)

type taggedEvent struct {
	Session string `json:"session"`
	agentTypes.Event
	Display string `json:"tool_display,omitempty"`
}

type wireEvent struct {
	agentTypes.Event
	Display string `json:"tool_display,omitempty"`
}

func skipEvent(ev agentTypes.Event) bool {
	switch ev.Type {
	case agentTypes.EventToolCall, agentTypes.EventToolResult, agentTypes.EventToolSkipped,
		agentTypes.EventToolCallStart, agentTypes.EventToolCallText, agentTypes.EventToolCallEnd,
		agentTypes.EventToolConfirm:
		return internalUtils.HideToolEvent(ev.ToolName, ev.ToolArgs)
	}
	return false
}

func toWire(ev agentTypes.Event) wireEvent {
	display := ""
	if ev.Type == agentTypes.EventToolCall {
		display = internalUtils.FormatToolEvent(ev.ToolName, ev.ToolArgs)
	}
	return wireEvent{Event: ev, Display: display}
}

func toTagged(sessionID string, ev agentTypes.Event) taggedEvent {
	return taggedEvent{
		Session: sessionID,
		Event:   ev,
		Display: toWire(ev).Display,
	}
}

type connectedFrame struct {
	Session string `json:"session,omitempty"`
	agentTypes.Event
	State string `json:"state,omitempty"`
}

func newConnectedFrame(sessionID string) connectedFrame {
	status := configStatus.Get(sessionID)
	return connectedFrame{
		Session: sessionID,
		Event:   agentTypes.Event{Type: agentTypes.EventConnected, Text: sessionID},
		State:   status.State,
	}
}

func StreamMultiLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		var sids []string
		for s := range strings.SplitSeq(strings.TrimSpace(c.Query("sessions")), ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sids = append(sids, s)
			}
		}

		withDaemon := internalUtils.IsLoopback(c.Request.RemoteAddr) && c.DefaultQuery("daemon", "1") != "0"
		if len(sids) == 0 && !withDaemon {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessions is required when daemon=0 or the caller is remote"})
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
				if skipEvent(ev) {
					continue
				}
				te := toTagged(sid, ev)
				if raw, err := json.Marshal(te); err == nil {
					fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
				}
			}
		}
		c.Writer.Flush()

		for _, sid := range sids {
			for _, ev := range OutstandingConfirms(sid) {
				if raw, err := json.Marshal(toTagged(sid, ev)); err == nil {
					fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
				}
			}
		}
		c.Writer.Flush()

		merged := make(chan taggedEvent, 1024)
		var fanInDropped atomic.Int64

		if withDaemon {
			sids = append(sids, daemonLogChannel)
		}

		for _, sid := range sids {
			sub := pubsub.Sub(sid, 1024)
			subs = append(subs, sub)

			go func(id string, s *pubsub.Subscriber) {
				for ev := range s.Events() {
					if skipEvent(ev) {
						continue
					}
					te := toTagged(id, ev)
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
