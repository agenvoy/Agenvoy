package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

type StdioClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	nextID     atomic.Int64
	inflightMu sync.Mutex
	inflight   map[int64]chan response
	closed     atomic.Bool
	writeMu    sync.Mutex
	readErr    error
	readDone   chan struct{}

	instructions string
}

func newStdioClient(ctx context.Context, cfg ServerConfig) (*StdioClient, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("StdinPipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("StdoutPipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd.Start: %w", err)
	}

	client := &StdioClient{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   bufio.NewReader(stdout),
		inflight: map[int64]chan response{},
		readDone: make(chan struct{}),
	}

	go client.read()

	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := client.initialize(initCtx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("client.initialize: %w", err)
	}
	return client, nil
}

func (c *StdioClient) read() {
	defer close(c.readDone)
	for {
		line, err := c.stdout.ReadBytes('\n')
		if len(line) > 0 {
			var res response
			if err := json.Unmarshal(line, &res); err == nil && res.ID != nil {
				c.inflightMu.Lock()
				ch, ok := c.inflight[*res.ID]
				if ok {
					delete(c.inflight, *res.ID)
				}
				c.inflightMu.Unlock()
				if ok {
					ch <- res
				}
			}
		}
		if err != nil {
			c.inflightMu.Lock()
			c.readErr = err

			pending := c.inflight
			c.inflight = map[int64]chan response{}
			c.inflightMu.Unlock()
			for _, ch := range pending {
				close(ch)
			}
			return
		}
	}
}

func (c *StdioClient) write(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	raw = append(raw, '\n')

	if _, err := c.stdin.Write(raw); err != nil {
		return fmt.Errorf("stdin.Write: %w", err)
	}
	return nil
}

func (c *StdioClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("client closed")
	}

	id := c.nextID.Add(1)
	ch := make(chan response, 1)
	c.inflightMu.Lock()
	c.inflight[id] = ch
	c.inflightMu.Unlock()

	if err := c.write(newRequest(id, method, params)); err != nil {
		c.inflightMu.Lock()
		delete(c.inflight, id)
		c.inflightMu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.inflightMu.Lock()
		delete(c.inflight, id)
		c.inflightMu.Unlock()
		return nil, ctx.Err()
	case res, ok := <-ch:
		if !ok {
			c.inflightMu.Lock()
			err := c.readErr
			c.inflightMu.Unlock()
			if err == nil {
				err = fmt.Errorf("connection closed")
			}
			return nil, err
		}
		if res.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", res.Error.Code, res.Error.Message)
		}
		return res.Result, nil
	}
}

func (c *StdioClient) notify(method string, params any) error {
	return c.write(newNotification(method, params))
}

func (c *StdioClient) initialize(ctx context.Context) error {
	result, err := c.call(ctx, "initialize", initializeParams())
	if err != nil {
		return err
	}
	c.instructions = parseInstructions(result)
	if err := c.notify("notifications/initialized", nil); err != nil {
		return fmt.Errorf("notifications/initialized: %w", err)
	}
	return nil
}

func (c *StdioClient) List(ctx context.Context) ([]Tool, error) {
	raw, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return resp.Tools, nil
}

func (c *StdioClient) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	raw, err := c.call(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}
	return extractText(raw)
}

func (c *StdioClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- c.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = c.cmd.Process.Kill()
			<-done
		}
	}
	return nil
}

func (c *StdioClient) Instructions() string {
	return c.instructions
}
