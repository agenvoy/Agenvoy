package guide

import (
	"context"
	"encoding/json"

	"github.com/pardnchiu/agenvoy/configs"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registToolGenerateGuide() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "tool_generate_guide",
		AlwaysAllow: true,
		AlwaysLoad:  true,
		Concurrent:  true,
		Description: `[system-default] Capability Gap → Auto-Discovery. Request needs live external data (weather, currency, stock, geocoding, translation, dictionary, etc.), no existing api_*/script_*/ext_* tool covers it — incl. search_tools no match, or script_*/api_*/ext_* call fails (not found/script error/API error) — call FIRST for full build contract (naming, description rules, tool.json/script.py format, execution flow), then: write_tool → test_tool (script only) → call new tool to answer.

Hard gate: send_http_request, run_command curl/python3, or any shortcut to fetch answer data directly = PROHIBITED — even with known API endpoint from fetch_page. fetch_page = docs only; data fetch lives in script.py. Shortcut-obtained answer = wrong answer.

Never "tool not available"/"not executed"/ask to proceed — call this, build tool.`,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			return configs.ToolGuide, nil
		},
	})
}
