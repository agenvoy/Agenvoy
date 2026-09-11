package exec

import (
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/note"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
)

const skillsHeader = "## Skills\n\n**`/<name>` = STRICT EXECUTION** — every SKILL.md step binding, tool calls required. Batch independent read-only steps same response; serialize only when a step needs an earlier result. FIRST step (often `ask_user`) before any other tool call — no skip-ahead even if input looks complete.\n\n`run_skill` path = advisory — consult, integrate fitting parts, ignore rest. Activate matching skill by intent even without explicit `/<name>`.\n\n"

func BuildSystemPrompts(workDir, extraSystemPrompt string, scanner *runtime.SkillScanner, sessionID string, allowAll bool, excludeSkills []string, model string) []provider.Message {
	var prompts []provider.Message
	if channel := channelSystemPrompt(sessionID); channel != "" {
		prompts = append(prompts, provider.Message{Role: "system", Content: channel})
	}
	prompts = append(prompts, provider.Message{Role: "system", Content: getSystemPrompt(workDir, extraSystemPrompt, scanner, sessionID, allowAll, excludeSkills, model)})
	if section := mcpInstructionsSection(); section != "" {
		prompts = append(prompts, provider.Message{Role: "system", Content: section})
	}
	return prompts
}

func channelSystemPrompt(sessionID string) string {
	switch {
	case strings.HasPrefix(sessionID, "tg-"):
		return configs.TelegramSystemPrompt
	case strings.HasPrefix(sessionID, "dc-"):
		return configs.DiscordSystemPrompt
	case strings.HasPrefix(sessionID, "ln-"):
		return configs.LineSystemPrompt
	}
	return ""
}

func mcpInstructionsSection() string {
	instructions := mcp.Manager().Instructions()
	if len(instructions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## MCP Server Instructions\n\nEach block is the operating contract its server declared; follow it for that server's tools.\n")
	for _, name := range slices.Sorted(maps.Keys(instructions)) {
		sb.WriteString("\n### ")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		sb.WriteString(instructions[name])
		sb.WriteString("\n")
	}
	return sb.String()
}

func getSystemPrompt(workDir string, extraSystemPrompt string, scanner *runtime.SkillScanner, sessionID string, allowAll bool, excludeSkills []string, model string) string {
	systemOS := host().os
	extraSection := strings.TrimSpace(extraSystemPrompt)

	template := filesystem.ApplyReplyLang(configs.SystemPrompt)

	skillsSection := ""
	if list := skillListBlock(scanner, excludeSkills); list != "" {
		skillsSection = skillsHeader + list
	}

	personaSection := ""
	if sessionID != "" {
		if err := configBot.Save(sessionID, "", "", false); err != nil {
			slog.Debug("sessionBot Save",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
		}
	}
	if name, body := configBot.Get(sessionID); body != "" {
		var sb strings.Builder
		sb.WriteString("## Bot Persona\n\n")
		if name != "" {
			fmt.Fprintf(&sb, "Your operating identity for this session is `%s`. Internalise the role description below and apply it to every reply unless an explicit user instruction overrides it.\n\n", name)
		} else {
			sb.WriteString("Internalise the role description below and apply it to every reply unless an explicit user instruction overrides it.\n\n")
		}
		sb.WriteString(body)
		sb.WriteString("\n\n---\n\n")
		personaSection = sb.String()
	}

	return strings.NewReplacer(
		"{{.SystemOS}}", systemOS,
		"{{.WorkPath}}", workDir,
		"{{.OutputDir}}", filesystem.OutputDir(),
		"{{.HostNote}}", hostNoteSection(),
		"{{.ReplyLanguage}}", filesystem.ReplyLangDirective(),
		"{{.BotPersona}}", personaSection,
		"{{.PermissionMode}}", buildPermissionModeSection(allowAll),
		"{{.AvailableSkills}}", skillsSection,
		"{{.AvailableNote}}", noteSection(),
		"{{.OfficialGuide}}", officialGuideSection(model),
		"{{.AgentGuide}}", agentGuideSection(workDir),
		"{{.ExtraSystemPrompt}}", extraSection,
	).Replace(template)
}

func noteSection() string {
	if len(note.List()) == 0 {
		return ""
	}
	return "\n## Note\n\nThe operator keeps notes in this workspace and they outrank anything else you find: every non-smalltalk request fires `find_note` with its key terms before you answer — in the same response as any RAG or web lookup, never in place of one — then whichever names look relevant are pulled in full with `mode=read`, those calls issued together. Answering from RAG, the web or memory without that call, or presenting a RAG/web file as one of these notes, is a failed turn.\n"
}

func agentGuideSection(workDir string) string {
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		path := filepath.Join(workDir, name)
		if !go_pkg_filesystem_reader.IsFile(path) {
			continue
		}

		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			slog.Debug("agent guide ReadText",
				slog.String("path", path),
				slog.String("error", err.Error()))
			continue
		}
		if content = strings.TrimSpace(content); content == "" {
			continue
		}
		return "`" + path + "`\n\n" + content
	}
	return ""
}

func officialGuideSection(model string) string {
	for key, one := range configs.OfficialGuides {
		if strings.Contains(model, key) {
			return strings.TrimSpace(one)
		}
	}
	return ""
}

func buildPermissionModeSection(allowAll bool) string {
	if allowAll {
		return strings.TrimRight(configs.PermissionAlwaysAllow, "\n")
	}
	return strings.TrimRight(configs.PermissionSingleConfirm, "\n")
}

func getChatCompletionsSystemPrompt(workDir string, scanner *runtime.SkillScanner, excludeSkills []string, model string) string {
	skillsSection := ""
	if list := skillListBlock(scanner, excludeSkills); list != "" {
		skillsSection = skillsHeader + list
	}

	return strings.NewReplacer(
		"{{.SystemOS}}", host().os,
		"{{.WorkPath}}", workDir,
		"{{.HostNote}}", hostNoteSection(),
		"{{.ReplyLanguage}}", filesystem.ReplyLangDirective(),
		"{{.AvailableSkills}}", skillsSection,
		"{{.AvailableNote}}", noteSection(),
		"{{.OfficialGuide}}", officialGuideSection(model),
	).Replace(filesystem.ApplyReplyLang(configs.ChatCompletionsSystemPrompt))
}

func BuildChatCompletionsSystemPrompts(workDir string, scanner *runtime.SkillScanner, excludeSkills []string, model string) []provider.Message {
	prompts := []provider.Message{{Role: "system", Content: getChatCompletionsSystemPrompt(workDir, scanner, excludeSkills, model)}}
	if section := mcpInstructionsSection(); section != "" {
		prompts = append(prompts, provider.Message{Role: "system", Content: section})
	}
	return prompts
}

func skillListBlock(scanner *runtime.SkillScanner, excludeSkills []string) string {
	if scanner == nil {
		return ""
	}
	names := scanner.List()
	if len(names) == 0 {
		return ""
	}

	excluded := make(map[string]bool, len(excludeSkills))
	for _, n := range excludeSkills {
		excluded[strings.TrimSpace(n)] = true
	}

	var b strings.Builder
	for _, n := range names {
		if excluded[n] {
			continue
		}
		desc := go_pkg_utils.TruncateString(scanner.Skills.ByName[n].Description, 512)
		b.WriteString("- ")
		b.WriteString(n)
		if desc != "" {
			b.WriteString(": ")
			b.WriteString(desc)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderActivation(s *skill.Skill) string {
	var b strings.Builder
	fmt.Fprintf(&b, "active skill: %s\nskill directory: %s\n\n---\n\n", s.Name, s.Path)
	if ext := strings.TrimSpace(configs.SkillExecution); ext != "" {
		b.WriteString(ext)
		b.WriteString("\n\n---\n\n")
	}
	if names := toolRegister.BuiltinNames(); len(names) > 0 {
		b.WriteString("### Built-in Tools\n\n")
		for _, name := range names {
			b.WriteString("- `")
			b.WriteString(name)
			b.WriteString("`\n")
		}
		b.WriteString("\n---\n\n")
	}
	b.WriteString(s.Resolved())
	return b.String()
}
