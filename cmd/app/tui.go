package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/app"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/runtime/tui"
	tuiHash "github.com/pardnchiu/agenvoy/internal/session/tui"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
)

func TUI() {
	// lipgloss.SetHasDarkBackground(true)

	tuiHash.New()

	if err := filesystem.Init(); err != nil {
		slog.Error("filesystem.Init",
			slog.String("error", err.Error()))
		return
	}
	if err := filesystem.LoadRuntime(); err != nil {
		slog.Warn("filesystem.LoadRuntime",
			slog.String("error", err.Error()))
	}
	if err := filesystem.OpenDB(); err != nil {
		slog.Error("filesystem.OpenDB",
			slog.String("error", err.Error()))
		return
	}
	defer filesystem.CloseDB()

	if err := historyStore.New(); err != nil {
		slog.Warn("historyStore.New",
			slog.String("error", err.Error()))
	}
	defer historyStore.Close()

	if err := usagelog.New(); err != nil {
		slog.Warn("usagelog.New",
			slog.String("error", err.Error()))
	}
	defer usagelog.Close()

	if !runtime.IsCurrent() {
		if err := app.SpawnDaemon(); err != nil {
			slog.Warn("daemon launch failed; running TUI without server",
				slog.String("error", err.Error()))
		}
	} else if r, err := runtime.Read(); err == nil && r.EnableClaudeCode {
		agentTypes.EnableClaudeCode = true
	}

	if err := torii.Init(filesystem.StoreDir); err != nil {
		slog.Error("store.Init",
			slog.String("error", err.Error()))
		return
	}
	defer torii.Close()

	if err := go_pkg_sandbox.CheckDependence(); err != nil {
		slog.Error("sandbox.CheckDependence",
			slog.String("error", err.Error()))
	}

	registry := app.NewAgentRegistry()
	scanner := runtime.NewSkillScanner()

	agents.Set(nil, app.SelectSummary(registry), registry, scanner)
	agents.SetRefresher(func() (agentTypes.Agent, agentTypes.Agent, agentTypes.AgentRegistry) {
		registry := app.NewAgentRegistry()
		return nil, app.SelectSummary(registry), registry
	})
	agents.MarkLoaded()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopWatcher := app.WatchConfig(ctx, func() {
		agents.Reload()
	})
	defer stopWatcher()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	go func() {
		<-quit
		cancel()
	}()

	if err := tui.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "tui.Run error: %v\n", err)
	}
}
