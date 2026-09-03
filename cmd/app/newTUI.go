package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"

	"github.com/charmbracelet/lipgloss"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/knowledge"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	chatbotTool "github.com/pardnchiu/agenvoy/internal/runtime/chatbot/tool"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/runtime/tui"
	sessionSummary "github.com/pardnchiu/agenvoy/internal/session/summary"
	tuiHash "github.com/pardnchiu/agenvoy/internal/session/tui"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
	"github.com/pardnchiu/agenvoy/internal/tools/subagent"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
)

func newTUI() {
	lipgloss.SetHasDarkBackground(true)

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
	if err := torii.Init(filesystem.StoreDir); err != nil {
		slog.Error("store.Init",
			slog.String("error", err.Error()))
		return
	}
	knowledge.Migrate()
	defer torii.Close()

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
	historyStore.MigrateAction()
	historyStore.MigrateSession()
	sessionSummary.MigrateCursor()

	if err := usagelog.New(); err != nil {
		slog.Warn("usagelog.New",
			slog.String("error", err.Error()))
	}
	defer usagelog.Close()
	usagelog.Migrate()

	imageTool.Register()
	audioTool.Register()
	chatbotTool.Register()

	if !runtime.IsCurrent() {
		if err := newDaemon(); err != nil {
			slog.Warn("daemon launch failed; running TUI without server",
				slog.String("error", err.Error()))
		}
	}

	fmt.Fprint(os.Stderr, "waiting for daemon to be ready...")
	shown := -1
	if err := waitDaemonReady(context.Background(), 3*time.Minute, func(elapsed time.Duration) {
		if sec := int(elapsed.Seconds()); sec != shown {
			shown = sec
			fmt.Fprintf(os.Stderr, "\r\033[Kwaiting for daemon to be ready... %ds", sec)
		}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "\ndaemon not reachable: %v\ncheck %s\n", err, filesystem.DaemonLogPath)
		return
	}
	fmt.Fprint(os.Stderr, "\r\033[K")

	if err := go_pkg_sandbox.CheckDependence(); err != nil {
		slog.Error("sandbox.CheckDependence",
			slog.String("error", err.Error()))
	}

	subagent.Register()

	mcpManager := initMCP(context.Background(), "")
	defer mcpManager.Close()
	mcp.SetManager(mcpManager)

	registry := buildAgentRegistry()
	scanner := runtime.NewSkillScanner()
	selectorBot := dispatcherSelector(registry)
	summaryBot := summarySelector(registry)

	agents.Set(selectorBot, summaryBot, registry, scanner)
	agents.SetRefresher(refreshHost)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
