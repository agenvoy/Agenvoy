package configs

import (
	"embed"
	"strings"
)

// * Prompts

//go:embed prompts/agent_selector.md
var AgentSelector string

//go:embed prompts/skill_execution.md
var SkillExecution string

//go:embed prompts/compact_exec_prompt.md
var CompactExecPrompt string

//go:embed prompts/old_history_extract_prompt.md
var OldHistoryExtractPrompt string

//go:embed prompts/compact_history_prompt.md
var CompactHistoryPrompt string

//go:embed prompts/summary_prompt.md
var SummaryPrompt string

//go:embed prompts/summary_context.md
var SummaryContext string

//go:embed prompts/followup.md
var FollowupPrompt string

//go:embed prompts/system_prompt/system_prompt.md
var SystemPrompt string

//go:embed prompts/system_prompt/chatcompletions_system_prompt.md
var ChatCompletionsSystemPrompt string

//go:embed prompts/default_session_prompt.md
var DefaultSessionPrompt string

//go:embed prompts/system_prompt/always_allow.md
var PermissionAlwaysAllow string

//go:embed prompts/system_prompt/single_confirm.md
var PermissionSingleConfirm string

//go:embed prompts/system_prompt/subagent_charter.md
var SubagentCharter string

//go:embed prompts/system_prompt/wsl_host.md
var WSLHost string

// * Prompts > systemPrompt > Chatbot

//go:embed prompts/system_prompt/chatbot/telegram_system_prompt.md
var TelegramSystemPrompt string

//go:embed prompts/system_prompt/chatbot/telegram_format.md
var TelegramFormat string

//go:embed prompts/system_prompt/chatbot/discord_system_prompt.md
var DiscordSystemPrompt string

//go:embed prompts/system_prompt/chatbot/discord_format.md
var DiscordFormat string

//go:embed prompts/system_prompt/chatbot/line_system_prompt.md
var LineSystemPrompt string

// * Prompts > Guide

//go:embed prompts/guide/tool_generate.md
var GuideToolGenerate string

//go:embed prompts/guide/tool_error.md
var GuideToolError string

//go:embed prompts/guide/rag_web.md
var GuideRAGWeb string

//go:embed prompts/guide/market_analysis.md
var GuideMarketAnalysis string

//go:embed prompts/guide/targeted_read.md
var GuideTargetedRead string

//go:embed prompts/guide/ask_user.md
var GuideAskUser string

//go:embed prompts/guide/subagent_dispatch.md
var GuideSubagentDispatch string

//go:embed prompts/guide/html_render.md
var GuideHtmlRender string

//go:embed prompts/guide/write_todo.md
var GuideWriteTodo string

// * Configs

//go:embed jsons/sensitive_path.json
var SensitivePath []byte

//go:embed jsons/exclude_list.json
var ExcludeList []byte

//go:embed jsons/read_only_command.json
var ReadOnlyCommand []byte

//go:embed jsons/tui_tools.json
var TUITools []byte

// * Official Guide

//go:embed prompts/official_guides/*.md
var officialGuideFS embed.FS

var OfficialGuideCommon, OfficialGuides = loadOfficialGuides()

func loadOfficialGuides() (string, map[string]string) {
	const dir = "prompts/official_guides"
	entries, err := officialGuideFS.ReadDir(dir)
	if err != nil {
		return "", nil
	}

	common := ""
	guides := make(map[string]string, len(entries))
	for _, entry := range entries {
		raw, err := officialGuideFS.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			continue
		}
		key := strings.TrimSuffix(entry.Name(), ".md")
		if key == "common" {
			common = string(raw)
			continue
		}
		guides[key] = string(raw)
	}
	return common, guides
}

const (
	PoisonRefusal     = "無法執行此操作"
	GuardrailSentinel = "[KARAPPO]"
)
