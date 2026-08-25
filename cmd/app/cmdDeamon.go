package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/record"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/knowledge"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	chatbotTool "github.com/pardnchiu/agenvoy/internal/runtime/chatbot/tool"
	"github.com/pardnchiu/agenvoy/internal/runtime/discord"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	"github.com/pardnchiu/agenvoy/internal/runtime/monitor"
	"github.com/pardnchiu/agenvoy/internal/runtime/routes"
	"github.com/pardnchiu/agenvoy/internal/runtime/routes/handler"
	"github.com/pardnchiu/agenvoy/internal/runtime/telegram"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	configStatus "github.com/pardnchiu/agenvoy/internal/session/config/status"
	tuiHash "github.com/pardnchiu/agenvoy/internal/session/tui"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
	geminiStt "github.com/pardnchiu/agenvoy/internal/tools/external/stt"
	"github.com/pardnchiu/agenvoy/internal/tools/subagent"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
)

var (
	discordMu          sync.Mutex
	discordBot         *discord.Bot
	lastDiscordEnabled bool
	lastDiscordToken   string

	telegramMu          sync.Mutex
	telegramBot         *telegram.Bot
	lastTelegramEnabled bool
	lastTelegramToken   string
)

func reloadDiscord(attempt int) {
	newToken := keychain.Get(discord.Key)
	newEnabled := false
	if cfg, err := config.Load(); err == nil && cfg != nil {
		newEnabled = cfg.DiscordEnabled
	}

	discordMu.Lock()
	defer discordMu.Unlock()

	if attempt == 0 && newEnabled == lastDiscordEnabled && newToken == lastDiscordToken {
		return
	}

	if discordBot != nil {
		_ = discord.Close(discordBot)
		discordBot = nil
	}

	if !newEnabled || newToken == "" {
		lastDiscordEnabled = newEnabled
		lastDiscordToken = newToken
		return
	}

	bot, err := discord.New()
	if err != nil {
		slog.Error("discord.New",
			slog.String("error", err.Error()),
			slog.Int("attempt", attempt))
		if attempt < reloadRetryMax {
			go func() {
				time.Sleep(reloadRetryDelay)
				reloadDiscord(attempt + 1)
			}()
		}
		return
	}
	lastDiscordEnabled = newEnabled
	lastDiscordToken = newToken
	discordBot = bot
}

const reloadRetryMax = 5
const reloadRetryDelay = 30 * time.Second

func reloadTelegram(attempt int) {
	newToken := keychain.Get(telegram.Key)
	newEnabled := false
	if cfg, err := config.Load(); err == nil && cfg != nil {
		newEnabled = cfg.TelegramEnabled
	}

	telegramMu.Lock()
	defer telegramMu.Unlock()

	if attempt == 0 && newEnabled == lastTelegramEnabled && newToken == lastTelegramToken {
		return
	}

	if telegramBot != nil {
		_ = telegram.Close(telegramBot)
		telegramBot = nil
	}

	if !newEnabled || newToken == "" {
		lastTelegramEnabled = newEnabled
		lastTelegramToken = newToken
		return
	}

	bot, err := telegram.New()
	if err != nil {
		slog.Error("telegram.New",
			slog.String("error", err.Error()),
			slog.Int("attempt", attempt))
		if attempt < reloadRetryMax {
			go func() {
				time.Sleep(reloadRetryDelay)
				reloadTelegram(attempt + 1)
			}()
		}
		return
	}
	lastTelegramEnabled = newEnabled
	lastTelegramToken = newToken
	telegramBot = bot
}

func loopbackListeners(port string) ([]net.Listener, error) {
	var listeners []net.Listener
	var firstErr error

	for _, host := range []string{"127.0.0.1", "[::1]"} {
		listener, err := net.Listen("tcp", host+":"+port)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("net.Listen",
				slog.String("addr", host+":"+port),
				slog.String("error", err.Error()))
			continue
		}
		listeners = append(listeners, listener)
	}

	if len(listeners) == 0 {
		return nil, firstErr
	}
	return listeners, nil
}

func cmdDaemon() {
	installDaemonSlog()
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
	if err := config.BackfillKeys(); err != nil {
		slog.Warn("config BackfillKeys",
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

	geminiStt.Register()
	imageTool.Register()
	chatbotTool.Register()

	if _, err := runtime.Init(); err != nil {
		if errors.Is(err, runtime.ErrAlreadyRunning) {
			slog.Error("daemon already running, aborting")
			return
		}
		slog.Warn("runtime.Init",
			slog.String("error", err.Error()))
	}
	configStatus.Reset()

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

	runtime.SetRunner(runSkill)
	if err := runtime.NewScheduler(); err != nil {
		slog.Error("runtime.SchedulerInit",
			slog.String("error", err.Error()))
	}
	defer runtime.StopScheduler()

	if err := runtime.AddSystemCron("*/15 * * * *", runSummaryCron); err != nil {
		slog.Warn("cron summaryGenerate",
			slog.String("error", err.Error()))
	}

	if err := runtime.AddSystemCron("*/30 * * * *", session.Clean); err != nil {
		slog.Warn("cron sessionClean",
			slog.String("error", err.Error()))
	}

	stopSchedulerWatcher := runtime.SchedulerWatcher(context.Background())
	defer stopSchedulerWatcher()

	stopWatcher := watchConfig(context.Background())
	defer stopWatcher()

	reloadDiscord(0)
	reloadTelegram(0)
	monitor.Start(context.Background())

	handler.StartWebConfirm(context.Background())

	route := routes.New()
	server := &http.Server{
		Addr:    "127.0.0.1:" + filesystem.Port,
		Handler: route,
	}

	listeners, err := loopbackListeners(filesystem.Port)
	if err != nil {
		slog.Error("net.Listen",
			slog.String("port", filesystem.Port),
			slog.String("error", err.Error()))
		return
	}

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
	slog.Info("⎯ daemon shutting down")

	discordMu.Lock()
	if discordBot != nil {
		_ = discord.Close(discordBot)
		discordBot = nil
	}
	discordMu.Unlock()
	telegramMu.Lock()
	if telegramBot != nil {
		_ = telegram.Close(telegramBot)
		telegramBot = nil
	}
	telegramMu.Unlock()
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

func watchConfig(ctx context.Context) func() {
	configDir := filepath.Dir(filesystem.ConfigPath)
	configBase := filepath.Base(filesystem.ConfigPath)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("fsnotify.NewWatcher",
			slog.String("error", err.Error()))
		return func() {}
	}
	if err := w.Add(configDir); err != nil {
		slog.Warn("fsnotify.Watcher Add",
			slog.String("dir", configDir),
			slog.String("error", err.Error()))
		_ = w.Close()
		return func() {}
	}

	stopCh := make(chan struct{})
	go func() {
		defer w.Close()
		var lastReload time.Time
		for {
			select {
			case <-stopCh:
				return
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if filepath.Base(ev.Name) != configBase {
					continue
				}
				if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Rename) {
					continue
				}
				if time.Since(lastReload) < 200*time.Millisecond {
					continue
				}
				lastReload = time.Now()
				if agents.Reload() {
					slog.Info("⎯ host reloaded: config change")
				}
				reloadDiscord(0)
				reloadTelegram(0)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				slog.Debug("fsnotify.Watcher",
					slog.String("error", err.Error()))
			}
		}
	}()
	return func() { close(stopCh) }
}

func runSkill(ctx context.Context, sessionID, skillName string) (string, error) {
	body, err := skill.GetSchedule(skillName)
	if err != nil {
		return "", fmt.Errorf("scheduler skill %q unreadable: %w", skillName, err)
	}
	sessionDir := filesystem.SessionDir(sessionID)
	if err := go_pkg_filesystem.CheckDir(sessionDir, true); err != nil {
		return "", err
	}
	if err := configBot.Save(sessionID, "", "", false); err != nil {
		slog.Debug("sessionBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	output, err := exec.ExecWithSubagent(exec.WithDcPushPrefix(ctx, skillName), body, sessionID, "", "", "", nil, "")
	if err != nil {
		return "", err
	}

	return output, nil
}
