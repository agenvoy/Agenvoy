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

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

const (
	providerQuotaTimeout = 10 * time.Second
	providerQuotaTTL     = 180
	quotaKeyPrefix       = "provider:quota:"
)

func readQuotaCache(id string) (provider.UsageRemaining, bool) {
	if !torii.Ready() {
		return provider.UsageRemaining{}, false
	}
	db := torii.DB(torii.DBToolCache)
	if db == nil {
		return provider.UsageRemaining{}, false
	}
	record, ok := db.Get(context.Background(), quotaKeyPrefix+id)
	if !ok {
		return provider.UsageRemaining{}, false
	}
	var entry provider.UsageRemaining
	if err := json.Unmarshal([]byte(record.Value()), &entry); err != nil {
		return provider.UsageRemaining{}, false
	}
	return entry, true
}

func writeQuotaCache(id string, entry provider.UsageRemaining) {
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

func ListProviderQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")

		ctx, cancel := context.WithTimeout(c.Request.Context(), providerQuotaTimeout)
		defer cancel()

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			quotas = make(map[string]gin.H, len(utils.QuotaSources))
		)

		for _, source := range utils.QuotaSources {
			if refresh {
				DropQuotaCache(source.ID)
			} else if cached, ok := readQuotaCache(source.ID); ok {
				quotas[source.ID] = gin.H{"remaining": cached, "cached": true}
				continue
			}

			wg.Add(1)
			go func(source utils.QuotaSource) {
				defer wg.Done()

				entry := gin.H{}
				cfg, err := agentKeychain.Config(ctx, source.ID)
				if err == nil {
					var remaining provider.UsageRemaining
					if remaining, err = source.Fn(ctx, cfg); err == nil {
						entry["remaining"] = remaining
						writeQuotaCache(source.ID, remaining)
					}
				}
				if err != nil {
					entry["error"] = err.Error()
				}

				mu.Lock()
				quotas[source.ID] = entry
				mu.Unlock()
			}(source)
		}
		wg.Wait()

		c.JSON(http.StatusOK, gin.H{"quota": quotas})
	}
}

func GetModelQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"quota": utils.ModelQuota(c.Request.Context(), strings.TrimSpace(c.Query("model")))})
	}
}
