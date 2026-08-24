package torii

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	toriidb "github.com/pardnchiu/ToriiDB/core/store"
)

const (
	SetDefault    = toriidb.SetDefault
	DBToolCache   = 0 // All tool cache
	DBSessionHist = 1 // Session conversation
	DBErrorMemory = 2 // Tool error
	DBKnowledge   = 3 // Knowledge
)

var (
	once     sync.Once
	initErr  error
	instance *toriidb.Store
)

func Init(path string) error {
	once.Do(func() {
		s, err := toriidb.New(path)
		if err != nil {
			initErr = fmt.Errorf("toriidb.New: %w", err)
			return
		}
		instance = s
	})
	return initErr
}

func Close() {
	if instance == nil {
		return
	}
	if err := instance.Close(); err != nil {
		slog.Warn("store.Close", slog.String("error", err.Error()))
	}
}

func DB(idx int) *toriidb.Session {
	session := instance.Session()
	if err := session.Select(idx); err != nil {
		slog.Error("store.DB.Select",
			slog.Int("index", idx),
			slog.String("error", err.Error()))
	}
	return session
}

func TTL(seconds int64) *int64 {
	ts := time.Now().Unix() + seconds
	return &ts
}
