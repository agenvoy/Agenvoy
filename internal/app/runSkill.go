package app

import (
	"context"
	"log/slog"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	execsubagent "github.com/pardnchiu/agenvoy/internal/agents/exec/subagent"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	configbot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func RunSkill(ctx context.Context, sessionID, skillName string) (string, error) {
	body, err := skill.GetSchedule(skillName)
	if err != nil {
		return "", utilsmsg.Trace("skill.GetSchedule", skillName, err)
	}
	sessionDir := filesystem.SessionDir(sessionID)
	if err := go_pkg_filesystem.CheckDir(sessionDir, true); err != nil {
		return "", err
	}
	if err := configbot.Save(sessionID, "", "", false); err != nil {
		slog.Debug("sessionBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	output, err := execsubagent.Exec(exec.WithSchedule(exec.WithBotPushPrefix(ctx, skillName)), body, sessionID, "", "", "", nil, false)
	if err != nil {
		return "", err
	}

	return output, nil
}
