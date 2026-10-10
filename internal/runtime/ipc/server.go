package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net"
	"os"
	"slices"
	"strings"
	"sync"

	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/sudo"
)

func Listen(ctx context.Context) (func(), error) {
	path := filesystem.DaemonSocketPath
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("os.Remove: %w", err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("os.Chmod: %w", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serve(ctx, conn)
		}
	}()

	return func() {
		listener.Close()
		os.Remove(path)
	}, nil
}

type conn struct {
	raw     net.Conn
	uuid    string
	writeMu sync.Mutex
	enc     *json.Encoder
	askMu   sync.Mutex
	asks    map[string]runtime.Request
}

var (
	ownerMu      sync.Mutex
	conns        = map[*conn]struct{}{}
	sessionOwner = map[string]string{}
)

func serve(ctx context.Context, raw net.Conn) {
	c := &conn{
		raw:  raw,
		enc:  json.NewEncoder(raw),
		asks: map[string]runtime.Request{},
	}
	ownerMu.Lock()
	conns[c] = struct{}{}
	ownerMu.Unlock()

	connCtx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		raw.Close()
		c.detach()
		c.resolveAll(errors.New("client disconnected"))
	}()
	go c.pumpAsks(connCtx)

	dec := json.NewDecoder(raw)
	for {
		var f Frame
		if err := dec.Decode(&f); err != nil {
			return
		}
		switch f.Type {
		case FrameRun:
			go c.run(ctx, f)
		case FrameReply:
			if f.Reply != nil {
				go c.reply(ctx, f.Reply)
			}
		}
	}
}

func (c *conn) write(f Frame) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.enc.Encode(f)
}

func (c *conn) pumpAsks(ctx context.Context) {
	notify, unregister := runtime.RegisterListener("cli-")
	defer unregister()

	for {
		for {
			id, req, ok := runtime.PickNextMatch("cli-", c.accepts)
			if !ok {
				break
			}
			ask := &Ask{
				ID:         id,
				Kind:       req.Kind,
				ToolName:   req.ToolName,
				ToolArgs:   req.ToolArgs,
				Restricted: req.Restricted,
			}
			if len(req.Restricted) > 0 {
				ask.NeedPassword = !sudo.Cached(ctx)
			}
			if req.AskUser != nil {
				ask.Questions = req.AskUser.Questions
			}
			c.askMu.Lock()
			c.asks[id] = req
			c.askMu.Unlock()
			c.write(Frame{Type: FrameAsk, SessionID: req.DeliverTo, Ask: ask})
		}

		select {
		case <-ctx.Done():
			return
		case <-notify:
		}
	}
}

func (c *conn) own(uuid, sessionID string) {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	c.uuid = uuid
	sessionOwner[sessionID] = uuid
}

func (c *conn) accepts(req runtime.Request) bool {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	owner, ok := sessionOwner[req.DeliverTo]
	if !ok {
		owner = sessionOwner[req.SessionID]
	}
	return owner == "" || owner == c.uuid
}

func (c *conn) detach() {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	delete(conns, c)
	if c.uuid == "" {
		return
	}
	for other := range conns {
		if other.uuid == c.uuid {
			return
		}
	}
	for sessionID, owner := range sessionOwner {
		if owner == c.uuid {
			delete(sessionOwner, sessionID)
		}
	}
}

func (c *conn) reply(ctx context.Context, r *Reply) {
	c.askMu.Lock()
	req, ok := c.asks[r.ID]
	delete(c.asks, r.ID)
	c.askMu.Unlock()
	if !ok {
		return
	}

	reply := runtime.Reply{
		Approve:   r.Approve,
		Remember:  r.Remember,
		AllowTurn: r.AllowTurn,
		Skip:      r.Skip,
		Reason:    r.Reason,
		Answers:   r.Answers,
	}
	if r.Abort {
		reply.Error = runtime.ErrUserCanceled
	}
	if reply.Approve && len(req.Restricted) > 0 {
		if err := sudo.Verify(ctx, r.Password); err != nil {
			reply = runtime.Reply{Reason: "system password verification failed: " + err.Error()}
		} else {
			reply.Verified = true
		}
	}
	runtime.Resolve(r.ID, reply)
}

func (c *conn) resolveAll(err error) {
	c.askMu.Lock()
	ids := slices.Collect(maps.Keys(c.asks))
	clear(c.asks)
	c.askMu.Unlock()
	for _, id := range ids {
		runtime.Resolve(id, runtime.Reply{Error: err})
	}
}

func (c *conn) run(ctx context.Context, f Frame) {
	if f.Rayload == nil {
		c.write(Frame{Type: FrameDone, Error: "run payload is required"})
		return
	}

	sessionID := strings.TrimSpace(f.SessionID)
	if sessionID == "" {
		sessionID = "temp-" + go_pkg_utils.UUID()
		if err := configBot.Save(sessionID, "", "", false); err != nil {
			c.write(Frame{Type: FrameDone, Error: err.Error()})
			return
		}
	}
	if f.UUID != "" {
		c.own(f.UUID, sessionID)
	}

	execCtx := agentTypes.WithOrigin(context.WithoutCancel(ctx), "cli-")
	content := strings.TrimSpace(f.Rayload.Input)
	data := exec.Prepare(exec.ExecuteMeta{
		Model:          f.Rayload.Model,
		Reasoning:      strings.TrimSpace(f.Rayload.Reasoning),
		WorkDir:        f.Rayload.WorkDir,
		Content:        content,
		Input:          content,
		SessionID:      sessionID,
		AllowAll:       f.Rayload.AllowAll,
		TUI:            true,
		PendingTask:    f.Rayload.PendingTask,
		HistoryContent: f.Rayload.HistoryContent,
	})

	events, wait := exec.Stream(execCtx, sessionID, 64, func(stream chan<- agentTypes.Event) error {
		return exec.Start(execCtx, data, stream)
	})
	for ev := range events {
		if ev.Type == agentTypes.EventTextDelta {
			continue
		}
		frame := Frame{Type: FrameEvent, SessionID: sessionID, Event: &ev}
		if ev.Err != nil {
			frame.Error = ev.Err.Error()
		}
		c.write(frame)
	}

	result := Frame{Type: FrameDone, SessionID: sessionID}
	if err := wait(); err != nil {
		result.Error = err.Error()
	}
	c.write(result)
}
