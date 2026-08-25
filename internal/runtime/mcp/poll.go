package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"
)

const (
	PollInterval = 60 * time.Second
	pollTimeout  = 30 * time.Second
)

func (m *MCP) Watch(ctx context.Context) {
	if m == nil {
		return
	}

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

func (m *MCP) poll(ctx context.Context) {
	m.mu.Lock()
	names := slices.Sorted(maps.Keys(m.clients))
	clients := make(map[string]Client, len(names))
	for _, name := range names {
		clients[name] = m.clients[name]
	}
	m.mu.Unlock()

	for _, name := range names {
		select {
		case <-ctx.Done():
			return
		default:
		}

		client := clients[name]
		pollCtx, cancel := context.WithTimeout(ctx, pollTimeout)
		tools, err := client.List(pollCtx)
		cancel()

		if err != nil {
			var sessionErr *SessionError
			if !errors.As(err, &sessionErr) {
				m.mu.Lock()
				m.lastError[name] = err.Error()
				m.mu.Unlock()
				continue
			}
			if ctx.Err() != nil {
				return
			}

			slog.Debug("mcp poll session lost, reconnecting",
				slog.String("server", name),
				slog.String("error", err.Error()))

			renewCtx, cancel := context.WithTimeout(ctx, pollTimeout)
			_, err := m.renewClient(renewCtx, name, client)
			cancel()
			if err != nil {
				slog.Warn("mcp poll renewClient",
					slog.String("server", name),
					slog.String("error", err.Error()))
			}
			continue
		}

		sum := toolsSignature(tools)

		m.mu.Lock()
		unchanged := m.toolsSum[name] == sum
		m.mu.Unlock()
		if unchanged {
			continue
		}

		slog.Info("mcp tool list changed, re-registering",
			slog.String("server", name),
			slog.Int("tools", len(tools)))

		refreshCtx, cancel := context.WithTimeout(ctx, pollTimeout)
		m.refresh(refreshCtx, name)
		cancel()
	}
}

func toolsSignature(list []Tool) string {
	sorted := slices.SortedFunc(slices.Values(list), func(a, b Tool) int {
		return strings.Compare(a.Name, b.Name)
	})

	hash := sha256.New()
	for _, tool := range sorted {
		hash.Write([]byte(tool.Name))
		hash.Write([]byte{0})
		hash.Write([]byte(tool.Description))
		hash.Write([]byte{0})
		hash.Write(tool.InputSchema)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
