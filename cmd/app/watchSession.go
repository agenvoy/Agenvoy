package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func watchSession(ctx context.Context) func() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("fsnotify.NewWatcher",
			slog.String("error", err.Error()))
		return func() {}
	}
	if err := w.Add(filesystem.SessionsDir); err != nil {
		slog.Warn("fsnotify.Watcher Add",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		_ = w.Close()
		return func() {}
	}

	stopCh := make(chan struct{})
	go func() {
		defer w.Close()
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
				if !ev.Has(fsnotify.Create) {
					continue
				}
				sessionID := filepath.Base(ev.Name)
				if strings.HasPrefix(sessionID, ".") {
					continue
				}
				if !go_pkg_filesystem_reader.IsDir(ev.Name) {
					continue
				}
				name, _ := configBot.Get(sessionID)
				slog.Info("⎯ session created",
					slog.String("session", sessionID),
					slog.String("name", name))
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
