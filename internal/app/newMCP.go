package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

func NewMCP(ctx context.Context, sessionID string) *mcp.MCP {
	manager, err := mcp.New(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		slog.Warn("mcp.New",
			slog.String("error", err.Error()))
		return nil
	}
	manager.RegisterAll(ctx)
	go manager.Watch(ctx)
	return manager
}
