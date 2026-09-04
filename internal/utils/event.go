package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

var eventLabel = map[string]string{
	"read_files":            "File",
	"run_command":           "Run",
	"open_file":             "Open",
	"download_file":         "Download",
	"pkg_manage":            "Package",
	"search_web":            "Search",
	"fetch_page":            "Fetch",
	"http_request":          "Send",
	"calculate":             "Calc",
	"test_tool":             "Test",
	"send_to_chatbot":       "Push",
	"list_chatbot":          "Chatbots",
	"mcp__kura__search_rag": "RAG",
	"mcp__kura__list_rag":   "RAG",
}

var hiddenEvent = map[string]bool{
	"ask_user":            true,
	"store_secret":        true,
	"write_todo":          true,
	"run_skill":           true,
	"reasoning_guide":     true,
	"find_tools":          true,
	"chat_history":        true,
	"find_knowledge":      true,
	"error_history":       true,
	"file_history":        true,
	"mcp__kura__list_rag": true,
}

func parseToolArgs(raw string) map[string]any {
	var argMap map[string]any
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &argMap)
	}
	return argMap
}

func hideToolEvent(name, mode string) bool {
	return name == "" || hiddenEvent[name] || mode == "list" || hiddenEvent[name+"/"+mode]
}

func HideToolEvent(name, raw string) bool {
	return hideToolEvent(name, eventMode(name, parseToolArgs(raw)))
}

var primaryArgKeys = []string{
	"q", "query", "keyword", "input", "text", "prompt",
	"symbol", "ticker", "db", "topic", "name", "url", "link", "path", "id",
}

var eventModeLabel = map[string]map[string]string{
	"find_files": {"list": "Files", "glob": "Glob Files", "search": "Search Files"},
	"edit_file":  {"write": "Write", "patch": "Edit", "remove": "Remove", "restore": "Restore"},
	"schedules":  {"list": "Schedules", "patch": "Edit", "remove": "Remove", "write": "Write"},
	"subagents":  {"invoke": "Subagents", "list": "Subagents"},
	"edit_skill": {"write": "Write Skill", "patch": "Edit Skill", "remove": "Remove Skill"},
	"edit_tool":  {"write": "Write Tool", "patch": "Edit Tool", "remove": "Remove Tool"},
}

func FormatToolEvent(name, raw string) string {
	argMap := parseToolArgs(raw)
	mode := eventMode(name, argMap)
	if hideToolEvent(name, mode) {
		return ""
	}

	arg := func(keys ...string) string {
		for _, key := range keys {
			if val, ok := argMap[key]; ok {
				if str, ok := val.(string); ok && strings.TrimSpace(str) != "" {
					return strings.TrimSpace(str)
				}
			}
		}
		return ""
	}

	label := eventLabel[name]
	if dic, ok := eventModeLabel[name]; ok {
		label = dic[mode]
	}
	if label == "" {
		label = shortToolName(name)
	}

	return label + "(" + eventArgs(name, mode, raw, argMap, arg) + ")"
}

func eventArgs(name, mode, raw string, argMap map[string]any, arg func(...string) string) string {
	switch name {
	case "find_files":
		return joinQueries(argMap)

	case "read_files":
		return joinPaths(argMap)

	case "edit_file":
		if path := arg("path"); path != "" {
			return path
		}
		if v, ok := argMap["version"].(float64); ok && v > 0 {
			return fmt.Sprintf("version %d", int64(v))
		}
		return arg("task_id")

	case "schedules":
		return arg("skill_name")

	case "subagents":
		if mode != "invoke" {
			return ""
		}
		if target := arg("self_id"); target != "" {
			return target
		}
		return arg("model")

	case "run_command":
		return joinArgv(raw)

	case "open_file":
		return arg("path")

	case "download_file":
		return arg("url", "link")

	case "pkg_manage":
		action := arg("action")
		if action == "" {
			action = "install"
		}
		if pkg := arg("package"); pkg != "" {
			return action + " " + pkg
		}
		return action

	case "fetch_page":
		return arg("link", "url")

	case "http_request":
		url := arg("url")
		if method := arg("method"); method != "" && url != "" {
			return strings.ToUpper(method) + " " + url
		}
		return url

	case "search_web":
		query := arg("query", "keyword")
		if timeRange := arg("time_range", "time"); query != "" && timeRange != "" {
			return query + " [" + timeRange + "]"
		}
		return query

	case "calculate":
		return joinStrings(argMap["expressions"])

	case "run_skill":
		return arg("skill", "name")

	case "edit_skill":
		return arg("path", "name")

	case "edit_tool", "test_tool":
		return arg("name")
	}

	if len(argMap) == 0 {
		return ""
	}
	if value := arg(primaryArgKeys...); value != "" {
		return value
	}
	return strings.Join(strings.Fields(raw), " ")
}

func eventMode(name string, argMap map[string]any) string {
	if mode, ok := argMap["mode"].(string); ok && strings.TrimSpace(mode) != "" {
		return strings.TrimSpace(mode)
	}

	switch name {
	case "find_files":
		mode := "list"
		for _, one := range asList(argMap["queries"]) {
			pattern, _ := one["pattern"].(string)
			if strings.TrimSpace(pattern) == "" {
				continue
			}
			if filePattern, _ := one["file_pattern"].(string); strings.TrimSpace(filePattern) != "" {
				return "search"
			}
			mode = "glob"
		}
		return mode

	case "edit_file":
		if targets, ok := argMap["targets"].([]any); ok && len(targets) > 0 {
			return "patch"
		}
		if content, _ := argMap["content"].(string); content != "" {
			return "write"
		}

	case "edit_skill", "edit_tool":
		if content, _ := argMap["content"].(string); content != "" {
			return "write"
		}
		if old, _ := argMap["old_string"].(string); old != "" {
			return "patch"
		}

	case "subagents":
		if task, _ := argMap["task"].(string); strings.TrimSpace(task) != "" {
			return "invoke"
		}
		return "list"

	case "schedules":
		return "list"

	}
	return ""
}

func shortToolName(name string) string {
	if rest, ok := strings.CutPrefix(name, "mcp__"); ok {
		if _, tool, found := strings.Cut(rest, "__"); found && tool != "" {
			return tool
		}
		return rest
	}
	if IsPlugTool(name) {
		return PlugToolBaseName(name)
	}
	return name
}

func asList(value any) []map[string]any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(list))
	for _, one := range list {
		if dic, ok := one.(map[string]any); ok {
			out = append(out, dic)
		}
	}
	return out
}

func joinQueries(argMap map[string]any) string {
	labels := make([]string, 0, 4)
	for _, one := range asList(argMap["queries"]) {
		dir, _ := one["dir"].(string)
		if dir = strings.TrimSpace(dir); dir == "" {
			dir = "."
		}
		loc := dir
		if filePattern, _ := one["file_pattern"].(string); strings.TrimSpace(filePattern) != "" {
			loc = strings.TrimRight(dir, "/") + "/" + strings.TrimSpace(filePattern)
		}
		if pattern, _ := one["pattern"].(string); strings.TrimSpace(pattern) != "" {
			loc += " [" + strings.TrimSpace(pattern) + "]"
		} else if recursive, ok := one["recursive"].(bool); ok && recursive {
			loc += " (recursive)"
		}
		labels = append(labels, loc)
	}
	return strings.Join(labels, ", ")
}

func joinStrings(value any) string {
	list, ok := value.([]any)
	if !ok {
		return ""
	}
	out := make([]string, 0, len(list))
	for _, one := range list {
		if str, ok := one.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return strings.Join(out, ", ")
}

func joinPaths(argMap map[string]any) string {
	paths := make([]string, 0, 4)
	for _, one := range asList(argMap["files"]) {
		if path, ok := one["path"].(string); ok && strings.TrimSpace(path) != "" {
			paths = append(paths, strings.TrimSpace(path))
		}
	}
	return strings.Join(paths, ", ")
}

func joinArgv(raw string) string {
	var params struct {
		Argv []string `json:"argv"`
	}
	if json.Unmarshal([]byte(raw), &params) != nil || len(params.Argv) == 0 {
		return ""
	}
	parts := make([]string, len(params.Argv))
	for i, one := range params.Argv {
		if one == "" || strings.ContainsAny(one, " \t\n\"'\\") {
			parts[i] = strconv.Quote(one)
			continue
		}
		parts[i] = one
	}
	return strings.Join(parts, " ")
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
