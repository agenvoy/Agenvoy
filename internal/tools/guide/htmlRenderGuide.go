package guide

import (
	"context"
	"encoding/json"

	"github.com/pardnchiu/agenvoy/configs"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registHtmlRenderGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "html_render_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `[system-default]
Read before writing any standalone HTML deliverable — report, dashboard, chart, map, 3D view.
Carries the CDN and import-map contract for d3 / three.js / MapLibre / Font Awesome, the 480 / 800 / 1024 responsive breakpoints, and the resize rules that canvas-backed views need.
Not preloaded in the system prompt.`,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			return configs.HtmlRenderGuide, nil
		},
	})
}
