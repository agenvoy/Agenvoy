package torii

import (
	"context"
	"encoding/json"
	"log/slog"
)

func CachedList(ctx context.Context, key string, ttl int64, fetch func() ([]string, error)) ([]string, error) {
	db := DB(DBToolCache)
	if Ready() {
		if record, ok := db.Get(ctx, key); ok {
			var list []string
			if json.Unmarshal([]byte(record.Value()), &list) == nil {
				return list, nil
			}
		}
	}

	list, err := fetch()
	if err != nil {
		return nil, err
	}

	if Ready() {
		raw, _ := json.Marshal(list)
		if err := db.Set(ctx, key, string(raw), TTL(ttl)); err != nil {
			slog.Debug("torii.CachedList", slog.String("key", key), slog.String("error", err.Error()))
		}
	}
	return list, nil
}

func DropKeys(ctx context.Context, keys ...string) {
	if !Ready() {
		return
	}
	DB(DBToolCache).Del(ctx, keys...)
}
