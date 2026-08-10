package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Client interface {
	List(ctx context.Context) ([]Tool, error)
	Call(ctx context.Context, name string, args map[string]any) (string, error)
	Instructions() string
	Close() error
}

type SessionError struct {
	Err error
}

func (e *SessionError) Error() string {
	return e.Err.Error()
}

func (e *SessionError) Unwrap() error {
	return e.Err
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type sdkClient struct {
	session *mcpsdk.ClientSession
}

func newClient(ctx context.Context, name string, cfg ServerConfig, onToolsChanged func()) (Client, error) {
	transport, err := cfg.Expand().toTransport(name)
	if err != nil {
		return nil, fmt.Errorf("server %q: %w", name, err)
	}

	var opts *mcpsdk.ClientOptions
	if onToolsChanged != nil {
		opts = &mcpsdk.ClientOptions{
			ToolListChangedHandler: func(context.Context, *mcpsdk.ToolListChangedRequest) {
				go onToolsChanged()
			},
		}
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "agenvoy", Version: "1.0.0"}, opts)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect %q: %w", name, err)
	}
	return &sdkClient{session: session}, nil
}

func (c *sdkClient) List(ctx context.Context) ([]Tool, error) {
	res, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, &SessionError{Err: fmt.Errorf("tools/list: %w", err)}
	}

	list := make([]Tool, 0, len(res.Tools))
	for _, tool := range res.Tools {
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			continue
		}
		list = append(list, Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: raw,
		})
	}
	return list, nil
}

func (c *sdkClient) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}

	res, err := c.session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", &SessionError{Err: fmt.Errorf("tools/call %q: %w", name, err)}
	}

	var sb strings.Builder
	for _, content := range res.Content {
		text, ok := content.(*mcpsdk.TextContent)
		if !ok {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(text.Text)
	}

	if res.IsError {
		if sb.Len() == 0 {
			return "", fmt.Errorf("tool error")
		}
		return "", fmt.Errorf("tool error: %s", sb.String())
	}
	if sb.Len() > 0 {
		return sb.String(), nil
	}

	raw, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}

func (c *sdkClient) Instructions() string {
	init := c.session.InitializeResult()
	if init == nil {
		return ""
	}
	return strings.TrimSpace(init.Instructions)
}

func (c *sdkClient) Close() error {
	return c.session.Close()
}
