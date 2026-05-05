package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

type HttpClient struct {
	url     string
	headers map[string]string

	httpClient *http.Client
	nextID     atomic.Int64
	closed     atomic.Bool

	once      sync.Once
	err       error
	sessionID string
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

func (c *HttpClient) init(ctx context.Context) error {
	c.once.Do(func() {
		params := map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "agenvoy",
				"version": "0.1.0",
			},
		}
		if _, err := c.call(ctx, "initialize", params); err != nil {
			c.err = fmt.Errorf("http initialize: %w", err)
			return
		}

		if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
			c.err = fmt.Errorf("notify initialized: %w", err)
			return
		}
	})
	return c.err
}

func (c *HttpClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("client closed")
	}

	id := c.nextID.Add(1)
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}

	if params != nil {
		body["params"] = params
	}

	result, err := go_pkg_http.POSTStream(ctx, c.httpClient, c.url, c.getHeaders(), body, "json")
	if err != nil {
		return nil, fmt.Errorf("go_pkg_http.POSTStream: %w", err)
	}
	defer result.Body.Close()

	if sid := result.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	if result.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(result.Body, 8192))
		return nil, fmt.Errorf("http %d: %s", result.StatusCode, strings.TrimSpace(string(raw)))
	}

	if strings.HasPrefix(result.Header.Get("Content-Type"), "text/event-stream") {
		return parseSSE(result.Body, id)
	}

	raw, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	return parse(raw, id)
}

func (c *HttpClient) notify(ctx context.Context, method string, params any) error {
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}

	if params != nil {
		body["params"] = params
	}

	result, err := go_pkg_http.POSTStream(ctx, c.httpClient, c.url, c.getHeaders(), body, "json")
	if err != nil {
		return fmt.Errorf("go_pkg_http.POSTStream: %w", err)
	}
	defer result.Body.Close()

	_, _ = io.Copy(io.Discard, result.Body)

	if result.StatusCode >= 400 {
		return fmt.Errorf("http %d", result.StatusCode)
	}
	return nil
}

func (c *HttpClient) getHeaders() map[string]string {
	headers := map[string]string{
		"Accept": "application/json, text/event-stream",
	}
	maps.Copy(headers, c.headers)

	if c.sessionID != "" {
		headers["Mcp-Session-Id"] = c.sessionID
	}
	return headers
}

func parse(raw []byte, id int64) (json.RawMessage, error) {
	var response Response
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", response.Error.Code, response.Error.Message)
	}

	if response.ID != nil && *response.ID != id {
		return nil, fmt.Errorf("response id mismatch: got %d want %d", *response.ID, id)
	}
	return response.Result, nil
}

func parseSSE(body io.Reader, id int64) (json.RawMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var data bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if data.Len() == 0 {
				continue
			}
			result, err := parse(data.Bytes(), id)
			data.Reset()
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
			continue
		}
		if payload, ok := strings.CutPrefix(line, "data:"); ok {
			payload = strings.TrimPrefix(payload, " ")
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(payload)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}
	if data.Len() > 0 {
		return parse(data.Bytes(), id)
	}
	return nil, fmt.Errorf("sse stream ended without rpc result")
}

func (c *HttpClient) List(ctx context.Context) ([]Tool, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}

	raw, err := c.call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return result.Tools, nil
}

func (c *HttpClient) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	if err := c.init(ctx); err != nil {
		return "", err
	}

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

func (c *HttpClient) Close() error {
	c.closed.Store(true)
	return nil
}
