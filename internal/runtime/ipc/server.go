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
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/sudo"
	"github.com/pardnchiu/agenvoy/internal/tools"
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

	tools.WorkDirChangeHook = notifyWorkDir

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
	writeMu sync.Mutex
	enc     *json.Encoder
	askMu   sync.Mutex
	asks    map[string]runtime.Request
	windows map[string]bool
	tasks   map[string]bool
}

var errDisconnected = errors.New("client disconnected")

var (
	runningMu   sync.Mutex
	runningConn = map[string]*conn{}
)

func notifyWorkDir(sessionID, dir string) {
	runningMu.Lock()
	c := runningConn[sessionID]
	runningMu.Unlock()
	if c != nil {
		c.write(Frame{Type: FrameWorkDir, SessionID: sessionID, Rayload: &Payload{WorkDir: dir}})
	}
}

func serve(ctx context.Context, raw net.Conn) {
	c := &conn{
		enc:     json.NewEncoder(raw),
		asks:    map[string]runtime.Request{},
		windows: map[string]bool{},
		tasks:   map[string]bool{},
	}

	connCtx, cancel := context.WithCancelCause(ctx)
	defer func() {
		cancel(errDisconnected)
		raw.Close()
		c.resolveAll(errDisconnected)
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
			go c.run(connCtx, f)
		case FrameCancel:
			c.askMu.Lock()
			own := c.tasks[f.TaskHash]
			c.askMu.Unlock()
			if !own {
				continue
			}
			cause := runtime.ErrUserCanceled
			if f.Pause {
				cause = nil
			}
			exec.Cancel(f.TaskHash, cause)
		case FrameSteer:
			if f.Rayload != nil {
				exec.AppendSteer(f.SessionID, f.Rayload.WindowHash, f.Rayload.Input)
			}
		case FramePending:
			go c.pending(connCtx, f)
		case FrameMCP:
			go c.mcp(connCtx, f)
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

func (c *conn) addWindow(windowHash string) {
	if windowHash == "" {
		return
	}
	c.askMu.Lock()
	c.windows[windowHash] = true
	c.askMu.Unlock()
}

func (c *conn) owns(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	c.askMu.Lock()
	defer c.askMu.Unlock()
	return c.windows[agentTypes.WindowHash(ctx)]
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
		err := sudo.Verify(ctx, r.Password)
		if err != nil {
			reply = runtime.Reply{Reason: "system password verification failed"}
			c.write(Frame{Type: FrameVerify, Error: err.Error()})
		} else {
			reply.Verified = true
			if r.Password != "" {
				c.write(Frame{Type: FrameVerify})
			}
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
