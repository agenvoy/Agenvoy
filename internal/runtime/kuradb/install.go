package kuradb

import (
	"context"
	"fmt"
	"os/exec"
	"slices"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

const (
	InstallURL = "https://agenvoy.com/scripts/kuradb.sh"
	ServerName = "kura"
	Binary     = "kura"
)

func Installed() bool {
	_, err := exec.LookPath(Binary)
	return err == nil
}

func Registered() (string, bool) {
	cfg, err := mcp.Load()
	if err != nil {
		return "", false
	}
	for name, server := range cfg.Servers {
		if server.Command == Binary && slices.Contains(server.Args, "mcp") {
			return name, true
		}
	}
	return "", false
}

func Reconnect(ctx context.Context, sessionID string) error {
	return mcp.Manager().Reconnect(ctx, sessionID)
}

func Register(ctx context.Context, sessionID string) error {
	if _, ok := Registered(); ok {
		return nil
	}

	cfg, err := mcp.Load()
	if err != nil {
		return fmt.Errorf("mcp.Load: %w", err)
	}
	cfg.Servers[ServerName] = mcp.ServerConfig{
		Command: Binary,
		Args:    []string{"mcp"},
	}
	if err := mcp.Save(cfg); err != nil {
		return fmt.Errorf("mcp.Save: %w", err)
	}
	return mcp.Manager().Reconnect(ctx, sessionID)
}
