package mcp

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

type MCP struct {
	mu      sync.Mutex
	clients map[string]Client
}

var (
	managerMu sync.RWMutex
	manager   *MCP
)

func SetManager(m *MCP) {
	managerMu.Lock()
	defer managerMu.Unlock()
	manager = m
}

func Manager() *MCP {
	managerMu.RLock()
	defer managerMu.RUnlock()
	return manager
}

type ServerInfo struct {
	Name      string
	Transport string
	Connected bool
}

func New(ctx context.Context, sessionID string) (*MCP, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	mcp := &MCP{
		clients: map[string]Client{},
	}

	for _, key := range slices.Sorted(maps.Keys(cfg.Servers)) {
		client, err := newClient(ctx, key, cfg.Servers[key], mcp.refresher(key))
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

func (m *MCP) Status(sessionID string) []ServerInfo {
	if m == nil {
		return nil
	}
	cfg, err := Load()
	if err != nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	list := make([]ServerInfo, 0, len(cfg.Servers))
	for _, name := range slices.Sorted(maps.Keys(cfg.Servers)) {
		s := cfg.Servers[name]
		transport := "stdio"
		if s.Expand().IsHTTP() {
			transport = "streamable-http"
		}
		_, connected := m.clients[name]
		list = append(list, ServerInfo{
			Name:      name,
			Transport: transport,
			Connected: connected,
		})
	}
	return list
}

type HealthInfo struct {
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (m *MCP) Instructions() map[string]string {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[string]string, len(m.clients))
	for _, name := range slices.Sorted(maps.Keys(m.clients)) {
		if text := strings.TrimSpace(m.clients[name].Instructions()); text != "" {
			out[name] = text
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (m *MCP) Health(ctx context.Context) []HealthInfo {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	names := slices.Sorted(maps.Keys(m.clients))
	clients := make(map[string]Client, len(names))
	for _, name := range names {
		clients[name] = m.clients[name]
	}
	m.mu.Unlock()

	list := make([]HealthInfo, 0, len(names))
	for _, name := range names {
		if _, err := clients[name].List(ctx); err != nil {
			list = append(list, HealthInfo{Name: name, OK: false, Error: err.Error()})
		} else {
			list = append(list, HealthInfo{Name: name, OK: true})
		}
	}
	return list
}

func (m *MCP) Reconnect(ctx context.Context, sessionID string) error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	for _, c := range m.clients {
		_ = c.Close()
	}
	m.clients = map[string]Client{}
	m.mu.Unlock()

	toolRegister.RemoveByPrefix("mcp__")

	cfg, err := Load()
	if err != nil {
		return err
	}

	m.mu.Lock()
	for _, key := range slices.Sorted(maps.Keys(cfg.Servers)) {
		client, err := newClient(ctx, key, cfg.Servers[key], m.refresher(key))
		if err != nil {
			slog.Warn("mcp reconnect newClient",
				slog.String("server", key),
				slog.String("error", err.Error()))
			continue
		}
		m.clients[key] = client
	}
	m.mu.Unlock()

	m.RegisterAll(ctx)
	return nil
}

func (m *MCP) refresher(name string) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.refresh(ctx, name)
	}
}

func (m *MCP) refresh(ctx context.Context, name string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	client, ok := m.clients[name]
	m.mu.Unlock()
	if !ok {
		return
	}

	tools, err := client.List(ctx)
	if err != nil {
		slog.Warn("mcp refresh client.List",
			slog.String("server", name),
			slog.String("error", err.Error()))
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clients[name] != client {
		return
	}

	toolRegister.RemoveByPrefix("mcp__" + name + "__")
	registered := 0
	for _, tool := range tools {
		def, ok := tool.getDef(name, client)
		if !ok {
			continue
		}
		toolRegister.Regist(def)
		registered++
	}
	slog.Info("mcp tools refreshed",
		slog.String("server", name),
		slog.Int("tools", registered))
}

func (m *MCP) RegisterAll(ctx context.Context) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, name := range slices.Sorted(maps.Keys(m.clients)) {
		client := m.clients[name]
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
