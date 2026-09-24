package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/app"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/record"
	"github.com/pardnchiu/agenvoy/internal/note"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	chatbotTool "github.com/pardnchiu/agenvoy/internal/runtime/chatbot/tool"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	"github.com/pardnchiu/agenvoy/internal/runtime/routes"
	"github.com/pardnchiu/agenvoy/internal/runtime/routes/handler"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/runtime/webapp"
	"github.com/pardnchiu/agenvoy/internal/session"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	sessionSummary "github.com/pardnchiu/agenvoy/internal/session/summary"
	tuiHash "github.com/pardnchiu/agenvoy/internal/session/tui"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
	"github.com/pardnchiu/agenvoy/internal/tools/subagent"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
)

func Daemon() {
	bootAt := time.Now()
	bootPhase := func(name string) {
		slog.Debug("boot phase",
			slog.String("phase", name),
			slog.Duration("elapsed", time.Since(bootAt)))
	}

	app.InstallDaemonLog()
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
	if err := record.TrimLog(); err != nil {
		slog.Warn("record TrimLog",
			slog.String("error", err.Error()))
	}
	if err := torii.Init(filesystem.StoreDir); err != nil {
		slog.Error("store.Init",
			slog.String("error", err.Error()))
		return
	}
	defer torii.Close()

	if n := interactive.ClearOnline(); n > 0 {
		slog.Info("cleared stale in-flight markers",
			slog.Int("count", n))
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
	historyStore.MigrateAction()
	historyStore.MigrateSession()
	sessionSummary.MigrateCursor()

	if err := usagelog.New(); err != nil {
		slog.Warn("usagelog.New",
			slog.String("error", err.Error()))
	}
	defer usagelog.Close()
	usagelog.Migrate()

	if err := note.New(); err != nil {
		slog.Warn("note.New",
			slog.String("error", err.Error()))
	}
	defer note.Close()
	note.Migrate()

	bootPhase("storage")

	imageTool.Register()
	audioTool.Register()
	chatbotTool.Register()

	if _, err := runtime.Init(); err != nil {
		if errors.Is(err, runtime.ErrAlreadyRunning) {
			slog.Error("daemon already running, aborting")
			return
		}
		slog.Warn("runtime.Init",
			slog.String("error", err.Error()))
	}

	if err := webapp.SyncAsset(context.Background()); err != nil {
		slog.Warn("webapp.SyncAsset",
			slog.String("error", err.Error()))
	}

	if path, err := webapp.Install(context.Background()); err != nil {
		slog.Warn("webapp.Install",
			slog.String("error", err.Error()))
	} else {
		slog.Debug("webapp.Install",
			slog.String("path", path))
	}

	if err := go_pkg_sandbox.CheckDependence(); err != nil {
		slog.Error("sandbox.CheckDependence",
			slog.String("error", err.Error()))
	}

	subagent.Register()

	defer func() {
		if m := mcp.Manager(); m != nil {
			m.Close()
		}
	}()

	agents.Set(nil, nil, agentTypes.AgentRegistry{}, runtime.NewSkillScanner())
	agents.SetRefresher(app.RefreshHost)

	go func() {
		mcp.SetManager(app.NewMCP(context.Background(), ""))
		bootPhase("mcp ready")
	}()

	runtime.SetRunner(app.RunSkill)
	if err := runtime.NewScheduler(); err != nil {
		slog.Error("runtime.SchedulerInit",
			slog.String("error", err.Error()))
	}
	defer runtime.StopScheduler()

	if err := runtime.AddSystemCron("*/15 * * * *", app.GenerateSummary); err != nil {
		slog.Warn("cron summaryGenerate",
			slog.String("error", err.Error()))
	}

	if err := runtime.AddSystemCron("*/30 * * * *", session.Clean); err != nil {
		slog.Warn("cron sessionClean",
			slog.String("error", err.Error()))
	}

	stopSchedulerWatcher := runtime.SchedulerWatcher(context.Background())
	defer stopSchedulerWatcher()

	stopWatcher := app.WatchConfig(context.Background(), func() {
		if agents.Reload() {
			slog.Debug("host reloaded: config change")
		}
		app.ReloadDiscord(0)
		app.ReloadTelegram(0)
		app.ReloadLine(0)
	})
	defer stopWatcher()

	stopSessionWatcher := app.WatchSession(context.Background())
	defer stopSessionWatcher()

	app.ReloadDiscord(0)
	app.ReloadTelegram(0)
	app.ReloadLine(0)

	handler.StartWebConfirm(context.Background())

	runtime.RegisterCancelNotifier(func(sessionID, taskHash, reason string) {
		event := agentTypes.Event{Type: agentTypes.EventCanceled, Text: reason}
		sessionLog.Record(sessionID, event)
		pubsub.Pub(sessionID, event)
	})

	route := routes.New()
	server := &http.Server{
		Addr:    "127.0.0.1:" + filesystem.Port,
		Handler: route,
	}

	listeners, err := app.LoopbackListeners(filesystem.Port)
	if err != nil {
		slog.Error("net.Listen",
			slog.String("port", filesystem.Port),
			slog.String("error", err.Error()))
		return
	}

	slog.Debug("listening",
		slog.String("port", filesystem.Port),
		slog.Duration("boot", time.Since(bootAt)))

	serveErr := make(chan error, len(listeners))
	for _, listener := range listeners {
		go func(l net.Listener) {
			if err := server.Serve(l); err != nil && err != http.ErrServerClosed {
				serveErr <- err
				return
			}
			serveErr <- nil
		}(listener)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case err := <-serveErr:
		if err != nil {
			slog.Error("server.Serve",
				slog.String("error", err.Error()))
		}
	}
	slog.Debug("daemon shutting down")

	app.CloseDiscord()
	app.CloseTelegram()
	app.CloseLine()
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}
	if err := runtime.Clear(); err != nil {
		slog.Warn("runtime.Clear",
			slog.String("error", err.Error()))
	}
}
