package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registSearchRag() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "search_rag",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Timeout:     15 * time.Second,
		Description: "[system-default] Search RAG knowledge base (keyword + semantic by default). mode=keyword for exact strings; mode=semantic for natural-language queries. Answer directly if results suffice.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"keyword", "semantic"},
					"description": "Narrow to a single search mode. Omit to run both keyword and semantic search together.",
				},
				"db": map[string]any{
					"type":        "string",
					"description": "Target RAG database name (default: agenvoy).",
					"default":     "agenvoy",
				},
				"q": map[string]any{
					"type":        "string",
					"description": "Search query.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Max chunks to return (1-100). Invalid values fall back to 10.",
					"default":     10,
				},
			},
			"required": []string{"db", "q"},
		},
		Handler: func(ctx context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode  string `json:"mode"`
				DB    string `json:"db"`
				Q     string `json:"q"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			db := strings.TrimSpace(params.DB)
			if db == "" {
				db = "agenvoy"
			}
			q := strings.TrimSpace(params.Q)
			if q == "" {
				return "", fmt.Errorf("q is required")
			}
			limit := params.Limit
			if limit < 1 || limit > 100 {
				limit = 10
			}

			target := strings.ToLower(strings.TrimSpace(params.Mode))
			switch target {
			case "", "keyword", "semantic":
			default:
				return "", fmt.Errorf("mode must be 'keyword' or 'semantic' (got %q)", params.Mode)
			}

			query := url.Values{}
			query.Set("db", db)
			query.Set("q", q)
			query.Set("limit", strconv.Itoa(limit))
			if target == "keyword" || target == "semantic" {
				query.Set("target", target)
			}
			return kuradbGet(ctx, "/api/search", query)
		},
	})
}
