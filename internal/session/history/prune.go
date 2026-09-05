package history

import (
	"context"
	"log/slog"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
)

const (
	vectorlessGrace = time.Hour
	vectorlessProbe = "session history vector probe"
)

func PruneVectorless() {
	if !torii.HasEmbedder() {
		return
	}

	ctx := context.Background()
	db := torii.DB(torii.DBSessionHist)

	keys := db.Keys(ctx, "*")
	if len(keys) == 0 {
		return
	}

	vectored, err := db.VSearch(ctx, vectorlessProbe, "*", len(keys))
	if err != nil {
		slog.Warn("PruneVectorless: VSearch",
			slog.String("error", err.Error()))
		return
	}

	has := make(map[string]struct{}, len(vectored))
	for _, one := range vectored {
		has[one] = struct{}{}
	}

	before := time.Now().Add(-vectorlessGrace).UnixNano()
	var dead []string
	for _, key := range keys {
		if _, ok := has[key]; ok {
			continue
		}
		ts, ok := getTimestamp(key)
		if !ok || ts >= before {
			continue
		}
		dead = append(dead, key)
	}
	if len(dead) == 0 {
		return
	}

	slog.Info("⎯ session history vectorless entries pruned",
		slog.Int("count", db.Del(ctx, dead...)))
}
