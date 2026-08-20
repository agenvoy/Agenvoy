package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func FormatToolEvent(name, raw string) string {
	if raw == "" || toolRegister.IsSystemUse(name) {
		return ""
	}

	var argMap map[string]any
	if err := json.Unmarshal([]byte(raw), &argMap); err != nil {
		return raw
	}
	if len(argMap) == 0 {
		return ""
	}

	arg := func(keys ...string) string {
		for _, key := range keys {
			if aryVal, ok := argMap[key]; ok {
				if str, ok := aryVal.(string); ok && strings.TrimSpace(str) != "" {
					return str
				}
			}
		}
		return ""
	}

	switch name {
	case "subagents":
		val := arg("name", "session_id")
		if val == "" {
			val = "subagent"
		}
		if model := arg("model"); model != "" {
			val = fmt.Sprintf("%s (%s)", val, model)
		}

		task := arg("task")
		if task == "" {
			return val
		}
		return fmt.Sprintf("%s: %s", val, strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(task))

	case "run_skill":
		if s := arg("skill", "name"); s != "" {
			return s
		}

	case "find_files":
		queries, ok := argMap["queries"].([]any)
		if !ok || len(queries) == 0 {
			break
		}
		labels := make([]string, 0, len(queries))
		for _, q := range queries {
			qm, ok := q.(map[string]any)
			if !ok {
				continue
			}
			dir, _ := qm["dir"].(string)
			dir = strings.TrimSpace(dir)
			if dir == "" {
				dir = "."
			}
			loc := dir
			if fp, _ := qm["file_pattern"].(string); strings.TrimSpace(fp) != "" {
				loc = strings.TrimRight(dir, "/") + "/" + fp
			}
			if pat, _ := qm["pattern"].(string); strings.TrimSpace(pat) != "" {
				loc += " [" + pat + "]"
			} else if r, ok := qm["recursive"].(bool); ok && r {
				loc += " (recursive)"
			}
			labels = append(labels, loc)
		}
		if len(labels) > 0 {
			return strings.Join(labels, ", ")
		}

	case "read_files":
		files, ok := argMap["files"].([]any)
		if !ok || len(files) == 0 {
			break
		}
		paths := make([]string, 0, len(files))
		for _, f := range files {
			fm, ok := f.(map[string]any)
			if !ok {
				continue
			}
			if p, ok := fm["path"].(string); ok && strings.TrimSpace(p) != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) > 0 {
			return strings.Join(paths, ", ")
		}

	case "edit_file":
		if val := arg("path", "pattern"); val != "" {
			return val
		}

	case "search_web":
		if val := arg("query", "keyword"); val != "" {
			if timeRange := arg("time_range", "time"); timeRange != "" {
				return fmt.Sprintf("%s (%s)", val, timeRange)
			}
			return val
		}

	case "fetch_yahoo_finance":
		if val := arg("symbol"); val != "" {
			if timeRange := arg("time_range"); timeRange != "" {
				return fmt.Sprintf("%s (%s)", val, timeRange)
			}
			return val
		}

	case "fetch_page":
		if val := arg("link", "url"); val != "" {
			return val
		}

	case "calculate":
		if val := arg("expression"); val != "" {
			return val
		}

	case "error_history":
		if val := arg("keyword", "hash", "symptom", "action"); val != "" {
			return val
		}

	case "chat_history":
		if val := arg("keyword", "query"); val != "" {
			return val
		}

	case "schedules":
		skill := arg("skill_name")
		t := arg("time")
		if skill != "" && t != "" {
			return fmt.Sprintf("%s %s", t, skill)
		}
		if skill != "" {
			return skill
		}

	case "run_command":
		var p struct {
			Argv []string `json:"argv"`
		}
		if err := json.Unmarshal([]byte(raw), &p); err != nil || len(p.Argv) == 0 {
			return raw
		}

		parts := make([]string, len(p.Argv))
		for i, arg := range p.Argv {
			if arg == "" || strings.ContainsAny(arg, " \t\n\"'\\") {
				parts[i] = strconv.Quote(arg)
			} else {
				parts[i] = arg
			}
		}
		return strings.Join(parts, " ")
	}
	return raw
}

var footerPrefixKeep = map[string]bool{
	"codex":      true,
	"copilot":    true,
	"grok-oauth": true,
}

func FormatEventFooter(duration time.Duration, model string, usage *provider.Usage) string {
	return formatEventFooter(duration, model, usage, "")
}

func FormatEventFooterContext(ctx context.Context, duration time.Duration, model string, usage *provider.Usage) string {
	return formatEventFooter(duration, model, usage, liveUsageSuffix(ctx, model))
}

func formatEventFooter(duration time.Duration, model string, usage *provider.Usage, modelSuffix string) string {
	var parts []string
	if duration > 0 {
		parts = append(parts, duration.Round(100*time.Millisecond).String())
	}

	if model = strings.TrimSpace(model); model != "" {
		if prefix, after, ok := strings.Cut(model, "@"); ok && !footerPrefixKeep[prefix] {
			model = after
		}
		if modelSuffix != "" {
			model += modelSuffix
		}
		parts = append(parts, model)
	}

	if usage != nil && (usage.Input > 0 || usage.CacheRead > 0 || usage.CacheCreate > 0 || usage.Output > 0) {
		totalInput := usage.Input + usage.CacheRead + usage.CacheCreate
		hitPct := 0
		if usage.CacheRead > 0 && totalInput > 0 {
			hitPct = int(float64(usage.CacheRead) / float64(totalInput) * 100)
		}
		if hitPct > 0 {
			parts = append(parts, fmt.Sprintf("↑ %s(%d%%) ↓ %s", go_pkg_utils.CompactNumber(totalInput), hitPct, go_pkg_utils.CompactNumber(usage.Output)))
		} else {
			parts = append(parts, fmt.Sprintf("↑ %s ↓ %s", go_pkg_utils.CompactNumber(totalInput), go_pkg_utils.CompactNumber(usage.Output)))
		}
	}
	return strings.Join(parts, " · ")
}

var footerUsageFn = map[string]func(context.Context, provider.Config) (float64, error){
	"codex":      openaicodex.Usage,
	"copilot":    copilot.Usage,
	"grok-oauth": grokoauth.Usage,
}

var footerBalanceFn = map[string]func(context.Context, provider.Config) (float64, error){
	"deepseek":   deepseek.Usage,
	"openrouter": openrouter.Usage,
}

func liveUsageSuffix(ctx context.Context, model string) string {
	prefix, _, ok := strings.Cut(strings.TrimSpace(model), "@")
	if !ok {
		return ""
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if fn, ok := footerUsageFn[prefix]; ok {
		cfg, err := agentKeychain.Config(ctx, prefix)
		if err != nil {
			return ""
		}
		remaining, err := fn(ctx, cfg)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("(%.0f%%)", remaining)
	}

	if fn, ok := footerBalanceFn[prefix]; ok {
		cfg, err := agentKeychain.Config(ctx, prefix)
		if err != nil {
			return ""
		}
		balance, err := fn(ctx, cfg)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("($%.2f)", balance)
	}

	return ""
}
