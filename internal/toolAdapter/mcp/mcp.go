package mcp

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

type MCP struct {
	mu      sync.Mutex
	clients map[string]Client
}

func New(ctx context.Context, sessionID string) (*MCP, error) {
	cfg, err := Read(sessionID)
	if err != nil {
		return nil, err
	}

	mcp := &MCP{
		clients: map[string]Client{},
	}

	for _, key := range slices.Sorted(maps.Keys(cfg.Servers)) {
		client, err := newClient(ctx, key, cfg.Servers[key])
		if err != nil {
			slog.Warn("newClient",
				slog.String("server", key),
				slog.String("error", err.Error()))
			continue
		}
		mcp.clients[key] = client
	}
	return mcp, nil
}

func (m *MCP) RegisterAll(ctx context.Context) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		tools, err := client.List(ctx)
		if err != nil {
			slog.Warn("client.List",
				slog.String("server", name),
				slog.String("error", err.Error()))
			continue
		}

		for _, tool := range tools {
			def, ok := tool.getDef(name, client)
			if !ok {
				slog.Warn("tool.getDef",
					slog.String("server", name),
					slog.String("tool", tool.Name))
				continue
			}
			toolRegister.Regist(def)
		}
	}
}

func (m *MCP) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		_ = client.Close()
	}
	m.clients = map[string]Client{}
}
