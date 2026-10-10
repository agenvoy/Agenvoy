package exec

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/guide"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func assignTurnContext(ctx context.Context, session *agentTypes.AgentSession, workDir string, allowAll bool, scanner *runtime.SkillScanner, excludeSkills []string, withSkills bool, toolNames []string) {
	var parts []string
	if !session.Stateless {
		parts = append(parts,
			"Work directory: `"+workDir+"` is authoritative this turn; ignore earlier ones in history. `run_command` already starts there — `cd` only to reach another directory: `run_command argv=[\"cd\", \"<path>\"]`.",
			buildPermissionModeSection(allowAll),
		)
	}
	if len(toolNames) > 0 {
		parts = append(parts, "Tools (call through run_tool): "+strings.Join(toolNames, ", "))
	}
	if withSkills {
		if list := skillListBlock(scanner, excludeSkills); list != "" {
			parts = append(parts, "Available skills:\n\n"+list)
		}
	}
	if !session.Stateless {
		if guideText := agentGuideSection(ctx, workDir); guideText != "" {
			parts = append(parts, guideText)
		}
	}
	if len(parts) == 0 {
		return
	}
	session.TurnContext = provider.Message{
		Role:    "user",
		Content: strings.Join(parts, "\n\n"),
	}
}

func agentGuideSection(ctx context.Context, workDir string) string {
	if !guide.IsEnabledCtx(ctx) {
		return ""
	}
	for _, name := range guide.Files {
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
		return "## External Agent Guide\n\n`" + path + "`\n\n" + content
	}
	return ""
}
