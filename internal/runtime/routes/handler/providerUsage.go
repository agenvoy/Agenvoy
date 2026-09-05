package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
)

const (
	providerUsageTimeout = 15 * time.Second
	providerUsageTTL     = 180
	usageKeyPrefix       = "provider:usage:"
)

type usageEntry struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
}

func readUsageCache(id string) (usageEntry, bool) {
	if !torii.Ready() {
		return usageEntry{}, false
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return usageEntry{}, false
	}
	record, ok := db.Get(context.Background(), usageKeyPrefix+id)
	if !ok {
		return usageEntry{}, false
	}
	var entry usageEntry
	if err := json.Unmarshal([]byte(record.Value()), &entry); err != nil {
		return usageEntry{}, false
	}
	return entry, true
}

func writeUsageCache(id string, entry usageEntry) {
	if !torii.Ready() {
		return
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	if err := db.Set(context.Background(), usageKeyPrefix+id, string(raw), torii.TTL(providerUsageTTL)); err != nil {
		slog.Debug("provider usage cache",
			slog.String("provider", id),
			slog.String("error", err.Error()))
	}
}

func DropUsageCache(id string) {
	if !torii.Ready() {
		return
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return
	}
	db.Del(context.Background(), usageKeyPrefix+id)
}

type usageSource struct {
	id   string
	kind string
	fn   func(context.Context, provider.Config) (float64, error)
}

var usageSources = []usageSource{
	{"codex", "percent", openaicodex.Usage},
	{"grok-oauth", "percent", grokoauth.Usage},
	{"copilot", "percent", copilot.Usage},
	{"openrouter", "balance", openrouter.Usage},
	{"deepseek", "balance", deepseek.Usage},
}

func ListProviderUsage() gin.HandlerFunc {
	return func(c *gin.Context) {
		refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")

		ctx, cancel := context.WithTimeout(c.Request.Context(), providerUsageTimeout)
		defer cancel()

		var (
			mu    sync.Mutex
			wg    sync.WaitGroup
			usage = make(map[string]gin.H, len(usageSources))
		)

		for _, source := range usageSources {
			if refresh {
				DropUsageCache(source.id)
			} else if cached, ok := readUsageCache(source.id); ok {
				usage[source.id] = gin.H{"kind": cached.Kind, "value": cached.Value, "cached": true}
				continue
			}

			wg.Add(1)
			go func(source usageSource) {
				defer wg.Done()

				entry := gin.H{"kind": source.kind}
				cfg, err := agentKeychain.Config(ctx, source.id)
				if err == nil {
					var value float64
					if value, err = source.fn(ctx, cfg); err == nil {
						entry["value"] = value
						writeUsageCache(source.id, usageEntry{Kind: source.kind, Value: value})
					}
				}
				if err != nil {
					entry["error"] = err.Error()
				}

				mu.Lock()
				usage[source.id] = entry
				mu.Unlock()
			}(source)
		}
		wg.Wait()

		c.JSON(http.StatusOK, gin.H{"usage": usage})
	}
}
