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
	providerQuotaTimeout = 15 * time.Second
	providerQuotaTTL     = 180
	quotaKeyPrefix       = "provider:quota:"
)

type quotaEntry struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
}

func readQuotaCache(id string) (quotaEntry, bool) {
	if !torii.Ready() {
		return quotaEntry{}, false
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return quotaEntry{}, false
	}
	record, ok := db.Get(context.Background(), quotaKeyPrefix+id)
	if !ok {
		return quotaEntry{}, false
	}
	var entry quotaEntry
	if err := json.Unmarshal([]byte(record.Value()), &entry); err != nil {
		return quotaEntry{}, false
	}
	return entry, true
}

func writeQuotaCache(id string, entry quotaEntry) {
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
	if err := db.Set(context.Background(), quotaKeyPrefix+id, string(raw), torii.TTL(providerQuotaTTL)); err != nil {
		slog.Debug("provider quota cache",
			slog.String("provider", id),
			slog.String("error", err.Error()))
	}
}

func DropQuotaCache(id string) {
	if !torii.Ready() {
		return
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return
	}
	db.Del(context.Background(), quotaKeyPrefix+id)
}

type quotaSource struct {
	id   string
	kind string
	fn   func(context.Context, provider.Config) (float64, error)
}

var quotaSources = []quotaSource{
	{"codex", "percent", openaicodex.Usage},
	{"grok-oauth", "percent", grokoauth.Usage},
	{"copilot", "percent", copilot.Usage},
	{"openrouter", "balance", openrouter.Usage},
	{"deepseek", "balance", deepseek.Usage},
}

func ListProviderQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")

		ctx, cancel := context.WithTimeout(c.Request.Context(), providerQuotaTimeout)
		defer cancel()

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			quotas = make(map[string]gin.H, len(quotaSources))
		)

		for _, source := range quotaSources {
			if refresh {
				DropQuotaCache(source.id)
			} else if cached, ok := readQuotaCache(source.id); ok {
				quotas[source.id] = gin.H{"kind": cached.Kind, "value": cached.Value, "cached": true}
				continue
			}

			wg.Add(1)
			go func(source quotaSource) {
				defer wg.Done()

				entry := gin.H{"kind": source.kind}
				cfg, err := agentKeychain.Config(ctx, source.id)
				if err == nil {
					var value float64
					if value, err = source.fn(ctx, cfg); err == nil {
						entry["value"] = value
						writeQuotaCache(source.id, quotaEntry{Kind: source.kind, Value: value})
					}
				}
				if err != nil {
					entry["error"] = err.Error()
				}

				mu.Lock()
				quotas[source.id] = entry
				mu.Unlock()
			}(source)
		}
		wg.Wait()

		c.JSON(http.StatusOK, gin.H{"quota": quotas})
	}
}
