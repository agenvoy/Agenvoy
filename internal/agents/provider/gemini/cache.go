package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	cacheAPI            = "https://generativelanguage.googleapis.com/v1beta/cachedContents"
	cacheMinTokens      = 4096
	cacheTTLSeconds     = 3600
	cacheExpirySafety   = 60 * time.Second
	cacheSweepThreshold = 64
)

type geminiCacheEntry struct {
	name       string
	prefixLen  int
	prefixHash string
	expiresAt  time.Time
}

type cacheCreateOutput struct {
	Name          string `json:"name"`
	UsageMetadata *struct {
		TotalTokenCount int `json:"totalTokenCount"`
	} `json:"usageMetadata,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *Agent) applyCache(ctx context.Context, systemPrompt string, messages []Content, tools []map[string]any) (cachedName string, tail []Content) {
	bucketKey := cacheBucketKey(a.model, systemPrompt, tools)

	a.cacheMu.Lock()
	entry := a.cacheStore[bucketKey]
	a.cacheMu.Unlock()

	if entry != nil && time.Now().Before(entry.expiresAt) &&
		entry.prefixLen <= len(messages) &&
		hashContents(messages[:entry.prefixLen]) == entry.prefixHash {
		return entry.name, messages[entry.prefixLen:]
	}

	if len(messages) < 2 {
		return "", messages
	}

	candidate := messages[:len(messages)-1]
	if estimateTokens(candidate, systemPrompt) < cacheMinTokens {
		return "", messages
	}

	created, err := a.createCache(ctx, candidate, systemPrompt, tools)
	if err != nil {
		slog.Warn("gemini cache create failed",
			slog.String("model", a.model),
			slog.String("error", err.Error()))
		return "", messages
	}

	a.cacheMu.Lock()
	if len(a.cacheStore) > cacheSweepThreshold {
		for k, v := range a.cacheStore {
			if time.Now().After(v.expiresAt) {
				delete(a.cacheStore, k)
			}
		}
	}
	a.cacheStore[bucketKey] = created
	a.cacheMu.Unlock()

	return created.name, messages[created.prefixLen:]
}

func (a *Agent) createCache(ctx context.Context, messages []Content, systemPrompt string, tools []map[string]any) (*geminiCacheEntry, error) {
	body := map[string]any{
		"model":    "models/" + a.model,
		"contents": messages,
		"ttl":      fmt.Sprintf("%ds", cacheTTLSeconds),
	}
	if systemPrompt != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": systemPrompt}},
		}
	}
	if len(tools) > 0 {
		body["tools"] = []map[string]any{
			{"functionDeclarations": tools},
		}
	}

	result, _, err := go_pkg_http.POST[cacheCreateOutput](ctx, a.httpClient, cacheAPI, map[string]string{
		"Content-Type":   "application/json",
		"x-goog-api-key": a.apiKey,
	}, body, "json")
	if err != nil {
		return nil, fmt.Errorf("http.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("cachedContents.create: %s", result.Error.Message)
	}
	if result.Name == "" {
		return nil, fmt.Errorf("cachedContents.create: empty name in response")
	}

	return &geminiCacheEntry{
		name:       result.Name,
		prefixLen:  len(messages),
		prefixHash: hashContents(messages),
		expiresAt:  time.Now().Add(cacheTTLSeconds*time.Second - cacheExpirySafety),
	}, nil
}

func cacheBucketKey(model, systemPrompt string, tools []map[string]any) string {
	raw, _ := json.Marshal(tools)
	sum := sha256.Sum256([]byte(model + "|" + systemPrompt + "|" + string(raw)))
	return hex.EncodeToString(sum[:])
}

func hashContents(messages []Content) string {
	raw, err := json.Marshal(messages)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func estimateTokens(messages []Content, systemPrompt string) int {
	raw, _ := json.Marshal(messages)
	return (len(raw) + len(systemPrompt)) / 4
}
