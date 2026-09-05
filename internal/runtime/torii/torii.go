package torii

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	go_pkg_filesystem_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"
	toriidb_daemon "github.com/pardnchiu/toriidb/core/daemon"
)

const openAIKeyName = "OPENAI_API_KEY"

const (
	DBToolCache   = 0 // All tool cache
	DBSessionHist = 1 // Session conversation
	DBErrorMemory = 2 // Tool error
	DBKnowledge   = 3 // * Legacy knowledge, read once by knowledge.Migrate
	DBOnline      = 3
)

type ScanOption = toriidb_daemon.ScanOption

var (
	once     sync.Once
	initErr  error
	instance *toriidb_daemon.Daemon
	embedder bool
)

func Init(path string) error {
	once.Do(func() {
		d, err := toriidb_daemon.New(path, go_pkg_filesystem_keychain.Get(openAIKeyName))
		if err != nil {
			initErr = fmt.Errorf("toriidb.New: %w", err)
			return
		}
		if err := d.Start(); err != nil {
			d.Close()
			initErr = fmt.Errorf("toriidb.Start: %w", err)
			return
		}
		instance = d

		// * the answer comes from whichever process owns the store, so a client
		// * gets the server's real state instead of guessing from its own keychain;
		// * on error assume it exists and let ErrNoEmbedder correct us per call
		ctx, cancel := bound(context.Background())
		defer cancel()

		embedder = true
		if has, err := d.HasEmbedder(ctx); err == nil {
			embedder = has
		} else {
			slog.Debug("torii.HasEmbedder", slog.String("error", err.Error()))
		}
	})
	return initErr
}

func Close() {
	if instance == nil {
		return
	}
	if err := instance.Close(); err != nil {
		slog.Warn("torii.Close", slog.String("error", err.Error()))
	}
}

func Ready() bool {
	return instance != nil
}

func HasEmbedder() bool {
	return instance != nil && embedder
}

func TTL(seconds int64) *int64 {
	ts := time.Now().Unix() + seconds
	return &ts
}
