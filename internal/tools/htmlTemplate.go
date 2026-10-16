package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/configs"
	toolregister "github.com/pardnchiu/agenvoy/internal/tools/register"
	tooltypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type htmlTemplate struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Desc     string `json:"desc"`
}

func registHTMLTemplate() {
	toolregister.Regist(toolregister.Def{
		Name:        "html_template",
		SystemUse:   true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Gallery of worked HTML page examples: list them, or read one's full HTML to copy.
Use while building an HTML deliverable (report, dashboard, chart, diagram) after reasoning_guide(topic=html_render).
Fetching any other web page → fetch_page.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "read"},
					"description": "list: every template's name, category, title, desc. read: one template's full HTML.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Template name exactly as mode=list returns it (e.g. '11-status-report', 'unknowns/06-interview'). Required for mode=read.",
				},
			},
			"required": []string{"mode"},
		},
		Timeout: 30 * time.Second,
		Handler: func(ctx context.Context, _ *tooltypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Mode string `json:"mode"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", utilsmsg.Wrap(utilsmsg.Invalid, "json.Unmarshal", "", err)
			}

			switch strings.TrimSpace(params.Mode) {
			case "list":
				return listHTMLTemplate(ctx)
			case "read":
				name := strings.TrimSpace(params.Name)
				if name == "" {
					return "", utilsmsg.New(utilsmsg.Invalid, "name", "required for mode=read")
				}
				return readHTMLTemplate(ctx, name)
			default:
				return "", utilsmsg.New(utilsmsg.Invalid, "mode", "unknown mode %q; available: list, read", params.Mode)
			}
		},
	})
}

func listHTMLTemplate(ctx context.Context) (string, error) {
	body, _, err := go_pkg_http.GET[struct {
		Templates []htmlTemplate `json:"templates"`
	}](ctx, http.DefaultClient, configs.ENDPOINT_HTML_TEMPLATE+"/list", nil)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "go_pkg_http.GET", "", err)
	}
	if len(body.Templates) == 0 {
		return "", utilsmsg.New(utilsmsg.Upstream, "html_template", "list is empty")
	}

	raw, err := json.Marshal(body.Templates)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "json.Marshal", "", err)
	}
	return string(raw), nil
}

func readHTMLTemplate(ctx context.Context, name string) (string, error) {
	body, _, err := go_pkg_http.GET[string](ctx, http.DefaultClient, configs.ENDPOINT_HTML_TEMPLATE+"/view/"+url.PathEscape(name), nil)
	if err != nil {
		return "", utilsmsg.New(utilsmsg.Upstream, "go_pkg_http.GET", "%s: %w; call mode=list for valid names", name, err)
	}
	return body, nil
}
