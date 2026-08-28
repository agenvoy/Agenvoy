package historyStore

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

var (
	frontmatterRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?(.*)$`)
	fieldRegex       = regexp.MustCompile(`(?m)^(\w+):\s*(.+)$`)
)

type legacyStatus struct {
	State string `json:"state"`
	Count int    `json:"count"`
}

type legacyBot struct {
	Name      string `json:"name"`
	Model     string `json:"model,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	Body      string `json:"body"`
}

func MigrateSession() {
	if conn == nil {
		return
	}

	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		slog.Warn("session migrate: ListDirs",
			slog.String("dir", filesystem.SessionsDir),
			slog.String("error", err.Error()))
		return
	}

	var migrated, broken int
	for _, one := range dirs {
		if strings.HasPrefix(one.Name, ".") {
			continue
		}
		switch migrateSessionRow(one.Name) {
		case 1:
			migrated++
		case -1:
			broken++
		}
	}

	if migrated+broken > 0 {
		slog.Info("⎯ session config migrated into sqlite",
			slog.Int("migrated", migrated),
			slog.Int("broken", broken))
	}
}

func migrateSessionRow(sessionID string) int {
	botPath := filesystem.BotPath(sessionID)
	legacyPath := filesystem.LegacyBotPath(sessionID)
	configPath := filesystem.SessionConfigPath(sessionID)
	statusPath := filesystem.StatusPath(sessionID)

	hasBot := go_pkg_filesystem_reader.Exists(botPath)
	hasLegacy := go_pkg_filesystem_reader.Exists(legacyPath)
	hasConfig := go_pkg_filesystem_reader.Exists(configPath)
	hasStatus := go_pkg_filesystem_reader.Exists(statusPath)
	if !hasBot && !hasLegacy && !hasConfig && !hasStatus {
		return 0
	}

	if hasStatus {
		migrateStatusRow(sessionID, statusPath)
	}
	if !hasBot && !hasLegacy && !hasConfig {
		return 1
	}

	row := SessionRow{SessionID: sessionID}
	if existing, ok, err := ReadSession(context.Background(), sessionID); err == nil && ok {
		row = existing
	}

	failed := false
	if hasBot {
		if bot, err := go_pkg_filesystem.ReadJSON[legacyBot](botPath); err == nil {
			applyBot(&row, bot)
		} else {
			slog.Warn("session migrate: bot.json",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
			failed = true
		}
	} else if hasLegacy {
		if bot, ok := parseLegacyBot(legacyPath); ok {
			applyBot(&row, bot)
		} else {
			failed = true
		}
	}

	if hasConfig {
		if dic, err := go_pkg_filesystem.ReadJSON[map[string]string](configPath); err == nil {
			applyChannel(&row, dic)
		} else {
			slog.Warn("session migrate: config.json",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
			failed = true
		}
	}

	if err := WriteSession(context.Background(), row); err != nil {
		slog.Warn("session migrate: WriteSession",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return -1
	}

	for _, path := range []string{botPath, legacyPath, configPath} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("session migrate: os.Remove",
				slog.String("path", path),
				slog.String("error", err.Error()))
		}
	}

	if failed {
		return -1
	}
	return 1
}

func migrateStatusRow(sessionID, statusPath string) {
	status, err := go_pkg_filesystem.ReadJSON[legacyStatus](statusPath)
	if err != nil {
		slog.Warn("session migrate: status.json",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	} else if _, ok, readErr := ReadState(context.Background(), sessionID); readErr == nil && !ok {
		if err := WriteState(context.Background(), StateRow{
			SessionID: sessionID,
			State:     status.State,
			InAction:  status.Count,
		}); err != nil {
			slog.Warn("session migrate: WriteState",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
			return
		}
	}

	if err := os.Remove(statusPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("session migrate: os.Remove",
			slog.String("path", statusPath),
			slog.String("error", err.Error()))
	}
}

func applyBot(row *SessionRow, bot legacyBot) {
	row.Name = bot.Name
	row.Model = bot.Model
	row.Reasoning = bot.Reasoning
	row.Rule = bot.Body
}

func applyChannel(row *SessionRow, dic map[string]string) {
	row.ChatID = dic["chat_id"]
	row.GuildID = dic["guild_id"]
	row.ChannelID = dic["channel_id"]
	row.UserID = dic["user_id"]
	if row.ChatID == "" {
		row.ChatID = dic["line_target"]
	}
}

func parseLegacyBot(path string) (legacyBot, bool) {
	data, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return legacyBot{}, false
	}

	bot := legacyBot{Body: strings.TrimSpace(data)}
	m := frontmatterRegex.FindStringSubmatch(data)
	if len(m) < 3 {
		return bot, true
	}

	bot.Body = strings.TrimSpace(m[2])
	for _, fm := range fieldRegex.FindAllStringSubmatch(m[1], -1) {
		switch fm[1] {
		case "name":
			bot.Name = strings.TrimSpace(fm[2])
		case "model":
			bot.Model = strings.TrimSpace(fm[2])
		case "reasoning":
			bot.Reasoning = strings.TrimSpace(fm[2])
		}
	}
	return bot, true
}
