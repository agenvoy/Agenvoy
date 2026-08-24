package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/auth"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	internalUtils "github.com/pardnchiu/agenvoy/internal/utils"
)

const webSessionPrefix = "chat-"

var (
	outstandingMu sync.Mutex
	outstanding   = map[string]map[string]agentTypes.Event{}
)

func trackConfirm(sessionID, requestID string, event agentTypes.Event) {
	outstandingMu.Lock()
	defer outstandingMu.Unlock()
	if outstanding[sessionID] == nil {
		outstanding[sessionID] = map[string]agentTypes.Event{}
	}
	outstanding[sessionID][requestID] = event
}

func OutstandingConfirms(sessionID string) []agentTypes.Event {
	outstandingMu.Lock()
	defer outstandingMu.Unlock()

	out := make([]agentTypes.Event, 0, len(outstanding[sessionID]))
	for id, event := range outstanding[sessionID] {
		if !runtime.EntryExists(id) {
			delete(outstanding[sessionID], id)
			continue
		}
		out = append(out, event)
	}
	return out
}

func restrictedOf(requestID string) []string {
	outstandingMu.Lock()
	defer outstandingMu.Unlock()
	for _, ids := range outstanding {
		if event, ok := ids[requestID]; ok {
			return event.Restricted
		}
	}
	return nil
}

func untrackConfirm(requestID string) {
	outstandingMu.Lock()
	defer outstandingMu.Unlock()
	for sessionID, ids := range outstanding {
		if _, ok := ids[requestID]; ok {
			delete(ids, requestID)
			if len(ids) == 0 {
				delete(outstanding, sessionID)
			}
			return
		}
	}
}

func StartWebConfirm(ctx context.Context) {
	notify, unregister := runtime.RegisterListener(webSessionPrefix)

	go func() {
		defer unregister()
		for {
			emitWebConfirms()
			select {
			case <-ctx.Done():
				return
			case <-notify:
			}
		}
	}()
}

func emitWebConfirms() {
	accept := func(r runtime.Request) bool { return r.Kind == runtime.KindToolConfirm }

	for {
		id, req, ok := runtime.PickNextMatch(webSessionPrefix, accept)
		if !ok {
			return
		}
		if req.Ctx != nil && req.Ctx.Err() != nil {
			runtime.Resolve(id, runtime.Reply{Error: req.Ctx.Err()})
			continue
		}
		event := agentTypes.Event{
			Type:       agentTypes.EventToolConfirm,
			ToolName:   req.ToolName,
			ToolArgs:   req.ToolArgs,
			ToolID:     id,
			Restricted: req.Restricted,
		}
		trackConfirm(req.SessionID, id, event)
		pubsub.Pub(req.SessionID, event)
	}
}

func ResolveToolConfirm() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.Param("request_id"))
		if requestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
			return
		}

		var body struct {
			Approve   bool   `json:"approve"`
			Remember  bool   `json:"remember,omitempty"`
			AllowTurn bool   `json:"allow_turn,omitempty"`
			Abort     bool   `json:"abort,omitempty"`
			Reason    string `json:"reason,omitempty"`
			Password  string `json:"password,omitempty"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !runtime.EntryExists(requestID) {
			c.JSON(http.StatusGone, gin.H{"error": "confirmation already resolved or expired"})
			return
		}

		approve := body.Approve && !body.Abort
		restricted := restrictedOf(requestID)
		verified := false

		if approve && len(restricted) > 0 {
			if !internalUtils.IsLoopback(c.Request.RemoteAddr) {
				c.JSON(http.StatusForbidden, gin.H{"error": "restricted approvals must come from this machine"})
				return
			}
			if err := auth.Verify(c.Request.Context(), body.Password); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error(), "restricted": restricted})
				return
			}
			verified = true
		}

		untrackConfirm(requestID)
		reply := runtime.Reply{
			Approve:   approve,
			Remember:  body.Remember,
			AllowTurn: body.AllowTurn,
			Reason:    strings.TrimSpace(body.Reason),
			Verified:  verified,
		}
		if body.Abort {
			reply.Error = errors.New("user stopped")
		}
		runtime.Resolve(requestID, reply)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
