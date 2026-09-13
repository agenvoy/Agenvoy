package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
)

func RunSkill(ctx context.Context, sessionID, skillName string) (string, error) {
	body, err := skill.GetSchedule(skillName)
	if err != nil {
		return "", fmt.Errorf("scheduler skill %q unreadable: %w", skillName, err)
	}
	sessionDir := filesystem.SessionDir(sessionID)
	if err := go_pkg_filesystem.CheckDir(sessionDir, true); err != nil {
		return "", err
	}
	if err := configBot.Save(sessionID, "", "", false); err != nil {
		slog.Debug("sessionBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	output, err := exec.ExecWithSubagent(exec.WithSchedule(exec.WithDcPushPrefix(ctx, skillName)), body, sessionID, "", "", "", nil, "", false)
	if err != nil {
		return "", err
	}

	return output, nil
}
