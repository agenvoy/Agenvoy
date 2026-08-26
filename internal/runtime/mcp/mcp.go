package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

type MCP struct {
	mu           sync.Mutex
	reconnectMu  sync.Mutex
	clients      map[string]Client
	lastError    map[string]string
	toolsSum     map[string]string
	instructions map[string]string
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
	Error     string
}

func New(ctx context.Context, sessionID string) (*MCP, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	mcp := &MCP{
		clients:      map[string]Client{},
		lastError:    map[string]string{},
		toolsSum:     map[string]string{},
		instructions: map[string]string{},
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
	cfg, err := Load()
	if err != nil {
		return nil
	}
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
	}

	list := make([]ServerInfo, 0, len(cfg.Servers))
	for _, name := range slices.Sorted(maps.Keys(cfg.Servers)) {
		s := cfg.Servers[name]
		transport := "stdio"
		if s.Expand().IsHTTP() {
			transport = "streamable-http"
		}
		info := ServerInfo{Name: name, Transport: transport}
		if m != nil {
			_, info.Connected = m.clients[name]
			info.Error = m.lastError[name]
		}
		list = append(list, info)
	}
	return list
}

func (m *MCP) Instructions() map[string]string {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[string]string, len(m.clients))
	for _, name := range slices.Sorted(maps.Keys(m.clients)) {
		text := strings.TrimSpace(m.clients[name].Instructions())
		if text != "" {
			m.instructions[name] = text
		} else {
			text = m.instructions[name]
		}
		if text != "" {
			out[name] = text
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
	m.lastError = map[string]string{}
	m.toolsSum = map[string]string{}
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

func (m *MCP) ReconnectServer(ctx context.Context, name string) error {
	if m == nil {
		return fmt.Errorf("no MCP manager")
	}
	cfg, err := Load()
	if err != nil {
		return err
	}
	server, ok := cfg.Servers[name]
	if !ok {
		return fmt.Errorf("server %q not found", name)
	}

	m.Disconnect(name)

	client, err := newClient(ctx, name, server, m.refresher(name))
	if err != nil {
		m.mu.Lock()
		m.lastError[name] = err.Error()
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	m.clients[name] = client
	m.mu.Unlock()

	m.refresh(ctx, name)
	return nil
}

func (m *MCP) Disconnect(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if client, ok := m.clients[name]; ok {
		_ = client.Close()
		delete(m.clients, name)
	}
	delete(m.lastError, name)
	delete(m.toolsSum, name)
	delete(m.instructions, name)
	m.mu.Unlock()

	toolRegister.RemoveByPrefix("mcp__" + name + "__")
}

func (m *MCP) Tools(ctx context.Context, name string) ([]Tool, error) {
	if m == nil {
		return nil, fmt.Errorf("no MCP manager")
	}
	m.mu.Lock()
	client, ok := m.clients[name]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("server %q is not connected", name)
	}
	return client.List(ctx)
}

func (m *MCP) Call(ctx context.Context, server, tool string, args map[string]any) (string, error) {
	if m == nil {
		return "", fmt.Errorf("no MCP manager")
	}

	m.mu.Lock()
	client, ok := m.clients[server]
	m.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("server %q is not connected", server)
	}

	out, err := client.Call(ctx, tool, args)
	var sessionErr *SessionError
	if err == nil || !errors.As(err, &sessionErr) {
		return out, err
	}
	if ctx.Err() != nil {
		return "", err
	}

	slog.Debug("mcp session lost, reconnecting",
		slog.String("server", server),
		slog.String("tool", tool),
		slog.String("error", err.Error()))

	client, reconnectErr := m.renewClient(ctx, server, client)
	if reconnectErr != nil {
		return "", fmt.Errorf("%w (reconnect %q: %v)", err, server, reconnectErr)
	}
	return client.Call(ctx, tool, args)
}

func (m *MCP) renewClient(ctx context.Context, server string, stale Client) (Client, error) {
	m.reconnectMu.Lock()
	defer m.reconnectMu.Unlock()

	m.mu.Lock()
	current, ok := m.clients[server]
	m.mu.Unlock()
	if ok && current != stale {
		return current, nil
	}

	if err := m.ReconnectServer(ctx, server); err != nil {
		return nil, err
	}

	m.mu.Lock()
	client, ok := m.clients[server]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("server %q is not connected", server)
	}
	return client, nil
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
		slog.Debug("mcp refresh client.List",
			slog.String("server", name),
			slog.String("error", err.Error()))
		m.mu.Lock()
		m.lastError[name] = err.Error()
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clients[name] != client {
		return
	}
	delete(m.lastError, name)

	toolRegister.RemoveByPrefix("mcp__" + name + "__")
	registered := 0
	for _, tool := range tools {
		def, ok := tool.getDef(name, m)
		if !ok {
			continue
		}
		toolRegister.Regist(def)
		registered++
	}
	m.toolsSum[name] = toolsSignature(tools)
	slog.Debug("mcp tools refreshed",
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
			slog.Debug("client.List",
				slog.String("server", name),
				slog.String("error", err.Error()))
			m.lastError[name] = err.Error()
			continue
		}
		delete(m.lastError, name)

		for _, tool := range tools {
			def, ok := tool.getDef(name, m)
			if !ok {
				slog.Debug("tool.getDef",
					slog.String("server", name),
					slog.String("tool", tool.Name))
				continue
			}
			toolRegister.Regist(def)
		}
		m.toolsSum[name] = toolsSignature(tools)
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
