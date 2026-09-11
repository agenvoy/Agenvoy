package configs

import (
	"embed"
	"strings"
)

// * Prompts

//go:embed prompts/skill_execution.md
var SkillExecution string

//go:embed prompts/summary_context.md
var SummaryContext string

//go:embed prompts/followup.md
var FollowupPrompt string

//go:embed prompts/system_prompt/agent_selector.md
var AgentSelector string

//go:embed prompts/system_prompt/memory/compact_exec_prompt.md
var CompactExecPrompt string

//go:embed prompts/system_prompt/memory/old_history_extract_prompt.md
var OldHistoryExtractPrompt string

//go:embed prompts/system_prompt/memory/compact_history_prompt.md
var CompactHistoryPrompt string

//go:embed prompts/system_prompt/memory/summary_prompt.md
var SummaryPrompt string

//go:embed prompts/system_prompt/system_prompt.md
var SystemPrompt string

//go:embed prompts/system_prompt/chatcompletions.md
var ChatCompletionsSystemPrompt string

//go:embed prompts/system_prompt/default_rule.md
var DefaultRule string

//go:embed prompts/system_prompt/permission/always_allow.md
var PermissionAlwaysAllow string

//go:embed prompts/system_prompt/permission/single_confirm.md
var PermissionSingleConfirm string

//go:embed prompts/system_prompt/subagent.md
var SubagentPrompt string

//go:embed prompts/system_prompt/wsl_host.md
var WSLHost string

// * Prompts > systemPrompt > Chatbot

//go:embed prompts/system_prompt/chatbot/telegram.md
var TelegramSystemPrompt string

//go:embed prompts/system_prompt/chatbot/discord.md
var DiscordSystemPrompt string

//go:embed prompts/system_prompt/chatbot/line.md
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

//go:embed prompts/guide/office.md
var GuideOffice string

// * Configs

//go:embed jsons/sensitive_path.json
var SensitivePath []byte

//go:embed jsons/exclude_list.json
var ExcludeList []byte

//go:embed jsons/read_only_command.json
var ReadOnlyCommand []byte

//go:embed jsons/tui_tools.json
var TUITools []byte

//go:embed jsons/reply_lang.json
var ReplyLang []byte

//go:embed jsons/local_compat.json
var LocalCompat []byte

// * Official Guide

//go:embed prompts/system_prompt/official_guides/*.md
var officialGuideFS embed.FS

var OfficialGuides = loadOfficialGuides()

func loadOfficialGuides() map[string]string {
	const dir = "prompts/system_prompt/official_guides"
	entries, err := officialGuideFS.ReadDir(dir)
	if err != nil {
		return nil
	}

	guides := make(map[string]string, len(entries))
	for _, entry := range entries {
		raw, err := officialGuideFS.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			continue
		}
		guides[strings.TrimSuffix(entry.Name(), ".md")] = string(raw)
	}
	return guides
}

const (
	PoisonRefusal     = "無法執行此操作"
	GuardrailSentinel = "[KARAPPO]"
)
