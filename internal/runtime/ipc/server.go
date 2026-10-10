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
	"sync"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
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
	uuids   map[string]struct{}
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
		raw:   raw,
		enc:   json.NewEncoder(raw),
		asks:  map[string]runtime.Request{},
		uuids: map[string]struct{}{},
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
	go c.askUser(connCtx)

	dec := json.NewDecoder(raw)
	for {
		var f Frame
		if err := dec.Decode(&f); err != nil {
			return
		}
		switch f.Type {
		case FrameRun:
			go c.run(ctx, f)
		case FrameCancel:
			exec.Cancel(f.TaskHash)
		case FramePending:
			go c.pending(ctx, f)
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

func (c *conn) own(uuid, sessionID string) {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	c.uuids[uuid] = struct{}{}
	sessionOwner[sessionID] = uuid
}

func (c *conn) accepts(req runtime.Request) bool {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	owner, ok := sessionOwner[req.DeliverTo]
	if !ok {
		owner = sessionOwner[req.SessionID]
	}
	if owner == "" {
		return true
	}
	_, mine := c.uuids[owner]
	return mine
}

func (c *conn) detach() {
	ownerMu.Lock()
	defer ownerMu.Unlock()
	delete(conns, c)
	alive := map[string]bool{}
	for other := range conns {
		for uuid := range other.uuids {
			alive[uuid] = true
		}
	}
	for sessionID, owner := range sessionOwner {
		if _, mine := c.uuids[owner]; mine && !alive[owner] {
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
