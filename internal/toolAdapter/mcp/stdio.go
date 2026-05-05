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
	inflight   map[int64]chan Response
	closed     atomic.Bool
	writeMu    sync.Mutex
	readErr    error
	readDone   chan struct{}
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
		inflight: map[int64]chan Response{},
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
			var response Response
			if err := json.Unmarshal(line, &response); err == nil && response.ID != nil {
				c.inflightMu.Lock()
				ch, ok := c.inflight[*response.ID]
				if ok {
					delete(c.inflight, *response.ID)
				}
				c.inflightMu.Unlock()
				if ok {
					ch <- response
				}
			}
		}
		if err != nil {
			c.inflightMu.Lock()
			c.readErr = err

			pending := c.inflight
			c.inflight = map[int64]chan Response{}
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
	ch := make(chan Response, 1)
	c.inflightMu.Lock()
	c.inflight[id] = ch
	c.inflightMu.Unlock()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	if err := c.write(req); err != nil {
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
	case response, ok := <-ch:
		if !ok {
			c.inflightMu.Lock()
			err := c.readErr
			c.inflightMu.Unlock()
			if err == nil {
				err = fmt.Errorf("connection closed")
			}
			return nil, err
		}
		if response.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", response.Error.Code, response.Error.Message)
		}
		return response.Result, nil
	}
}

func (c *StdioClient) notify(method string, params any) error {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		msg["params"] = params
	}
	return c.write(msg)
}

func (c *StdioClient) initialize(ctx context.Context) error {
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "agenvoy",
			"version": "0.1.0",
		},
	}
	if _, err := c.call(ctx, "initialize", params); err != nil {
		return err
	}
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
