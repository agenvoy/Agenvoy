package torii

import (
	"context"
	"errors"
	"log/slog"
	"time"

	toriidb_daemon "github.com/pardnchiu/toriidb/core/daemon"
)

const callTimeout = 10 * time.Second

var (
	errNotReady = errors.New("toriidb is not initialised")

	// * the store's own sentinel, so errors.Is holds whether the refusal came
	// * from this process or across the socket
	ErrNoEmbedder = toriidb_daemon.ErrNoEmbedder
)

type Entry struct {
	Key string `json:"key"`
	Val string `json:"value"`
}

func (e *Entry) Value() string {
	return e.Val
}

type Session struct {
	db int
}

func DB(idx int) *Session {
	return &Session{db: idx}
}

func bound(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, callTimeout)
}

func (s *Session) Get(ctx context.Context, key string) (*Entry, bool) {
	if instance == nil {
		return nil, false
	}
	ctx, cancel := bound(ctx)
	defer cancel()

	record, err := instance.Get(ctx, s.db, key)
	if err != nil {
		return nil, false
	}
	value, err := record.Value()
	if err != nil {
		slog.Debug("torii.Get",
			slog.String("key", key),
			slog.String("error", err.Error()))
		return nil, false
	}
	return &Entry{Key: key, Val: value}, true
}

func (s *Session) Keys(ctx context.Context, pattern string) []string {
	if instance == nil {
		return nil
	}
	ctx, cancel := bound(ctx)
	defer cancel()

	list, err := instance.Keys(ctx, s.db, pattern, 0, 0)
	if err != nil {
		slog.Debug("torii.Keys", slog.String("error", err.Error()))
		return nil
	}
	return list.Keys
}

func (s *Session) Scan(ctx context.Context, pattern string, opt ScanOption) []Entry {
	if instance == nil {
		return nil
	}
	ctx, cancel := bound(ctx)
	defer cancel()

	records, err := instance.Entries(ctx, s.db, pattern, &opt)
	if err != nil {
		slog.Debug("torii.Scan", slog.String("error", err.Error()))
		return nil
	}

	list := make([]Entry, 0, len(records))
	for _, one := range records {
		value, err := one.Value()
		if err != nil {
			continue
		}
		list = append(list, Entry{Key: one.Key, Val: value})
	}
	return list
}

func (s *Session) VSearch(ctx context.Context, text, pattern string, k int) ([]string, error) {
	if instance == nil {
		return nil, errNotReady
	}
	if !embedder {
		return nil, ErrNoEmbedder
	}

	ctx, cancel := bound(ctx)
	defer cancel()

	return instance.VSearch(ctx, s.db, text, pattern, k)
}

func (s *Session) Set(ctx context.Context, key, value string, expireAt *int64) error {
	if instance == nil {
		return errNotReady
	}

	ctx, cancel := bound(ctx)
	defer cancel()

	return instance.Set(ctx, s.db, key, value, expireAt)
}

func (s *Session) SetVector(ctx context.Context, key, value string, expireAt *int64) error {
	if instance == nil {
		return errNotReady
	}

	ctx, cancel := bound(ctx)
	defer cancel()

	if !embedder {
		return instance.Set(ctx, s.db, key, value, expireAt)
	}

	// * only the store refusing the vector falls back; every other failure is a
	// * real write error and must reach the caller
	err := instance.SetVector(ctx, s.db, key, value, expireAt)
	if !errors.Is(err, ErrNoEmbedder) {
		return err
	}

	slog.Warn("torii.SetVector falling back to plain write",
		slog.String("key", key),
		slog.String("error", err.Error()))
	return instance.Set(ctx, s.db, key, value, expireAt)
}

func (s *Session) Expire(ctx context.Context, key string, seconds int64) error {
	return s.ExpireMany(ctx, []string{key}, seconds)
}

func (s *Session) ExpireMany(ctx context.Context, keys []string, seconds int64) error {
	if instance == nil {
		return errNotReady
	}
	ctx, cancel := bound(ctx)
	defer cancel()

	_, err := instance.Expire(ctx, s.db, seconds, keys...)
	return err
}

func (s *Session) Del(ctx context.Context, keys ...string) int {
	if instance == nil {
		return 0
	}
	ctx, cancel := bound(ctx)
	defer cancel()

	count, err := instance.Del(ctx, s.db, keys...)
	if err != nil {
		slog.Debug("torii.Del", slog.String("error", err.Error()))
		return 0
	}
	return count
}
