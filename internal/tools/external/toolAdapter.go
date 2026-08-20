package external

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/agenvoy/internal/tools/external/searchWeb"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	searchWeb.Register()
	toolRegister.Regist(toolRegister.Def{
		Name:        "http_request",
		AlwaysAllow: false,
		Concurrent:  true,
		Description: `Sends one HTTP request to any URL and returns status, headers and body — GET through DELETE, multipart upload included.
Use for 打 API / 呼叫端點 / POST 一份資料, when no api_* tool covers that endpoint.
HTML meant to be read → fetch_page; a binary file → download_file; an endpoint you will call again → build an api_* tool with edit_tool.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "Full URL (e.g. 'https://api.example.com/v1/items').",
				},
				"method": map[string]any{
					"type":        "string",
					"description": "HTTP method.",
					"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
					"default":     "GET",
				},
				"headers": map[string]any{
					"type":        "object",
					"description": "Headers (e.g. {\"Authorization\": \"Bearer ...\"}).",
					"default":     map[string]any{},
				},
				"body": map[string]any{
					"type":        "object",
					"description": "Request body (POST/PUT/PATCH). content_type=json/form: flat key-value object. content_type=multipart: {\"fields\":{key:value,...},\"files\":[{\"name\":\"field\",\"path\":\"/abs/path\",\"content_type\":\"application/gzip\"},...]}. File path must be absolute; binary read from disk.",
					"default":     map[string]any{},
				},
				"content_type": map[string]any{
					"type":        "string",
					"description": "Body encoding. multipart for file uploads (binary).",
					"enum":        []string{"json", "form", "multipart"},
					"default":     "json",
				},
				"timeout": map[string]any{
					"type":        "integer",
					"description": "Timeout seconds (max 300). Use 120+ for compute-heavy APIs.",
					"default":     30,
				},
			},
			"required": []string{"url"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				URL         string            `json:"url"`
				Method      string            `json:"method"`
				Headers     map[string]string `json:"headers"`
				Body        map[string]any    `json:"body"`
				ContentType string            `json:"content_type"`
				Timeout     int               `json:"timeout"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			return sendHTTPRequest(ctx, params.URL, params.Method, params.Headers, params.Body, params.ContentType, params.Timeout)
		},
	})
}
