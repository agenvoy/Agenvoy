package configBot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
)

const (
	DefaultModel     = historyStore.DefaultModel
	DefaultReasoning = historyStore.DefaultReasoning
)

func read(sessionID string) (historyStore.SessionRow, bool) {
	if sessionID == "" {
		return historyStore.SessionRow{}, false
	}

	row, ok, err := historyStore.ReadSession(context.Background(), sessionID)
	if err != nil {
		slog.Warn("historyStore.ReadSession",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return historyStore.SessionRow{}, false
	}
	row.SessionID = sessionID
	return row, ok
}

func write(row historyStore.SessionRow) error {
	if err := historyStore.WriteSession(context.Background(), row); err != nil {
		return fmt.Errorf("historyStore.WriteSession: %w", err)
	}
	return nil
}

func Get(sessionID string) (name, body string) {
	row, _ := read(sessionID)
	return row.Name, row.Rule
}

func GetPersona(sessionID string) (selfID, name, body string) {
	row, _ := read(sessionID)
	return row.SelfID, row.Name, row.Rule
}

func SavePersona(sessionID, selfID, name, body string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	if err := historyStore.ValidSelfID(selfID); err != nil {
		return err
	}
	if name == "" {
		name = sessionID
	}
	if body == "" {
		body = configs.DefaultSessionPrompt
	}

	row, _ := read(sessionID)
	row.SessionID = sessionID
	row.SelfID = selfID
	row.Name = name
	row.Rule = body
	return write(row)
}

func GetModel(sessionID string) (model, reasoning string) {
	row, _ := read(sessionID)
	model = row.Model
	reasoning = row.Reasoning
	if model == "" {
		model = DefaultModel
	}
	if reasoning == "" {
		reasoning = DefaultReasoning
	}
	return model, reasoning
}

func SetModel(sessionID, model, reasoning string) {
	row, _ := read(sessionID)
	if row.SessionID == "" {
		return
	}
	if model != "" {
		row.Model = model
	}
	if reasoning != "" {
		row.Reasoning = reasoning
	}
	if err := write(row); err != nil {
		slog.Warn("configBot SetModel",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}

func SetChannel(sessionID string, chatID, guildID, channelID, userID string) error {
	row, _ := read(sessionID)
	if row.SessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	row.ChatID = chatID
	row.GuildID = guildID
	row.ChannelID = channelID
	row.UserID = userID
	return write(row)
}

func GetChannel(sessionID string) (chatID, guildID, channelID, userID string) {
	row, _ := read(sessionID)
	return row.ChatID, row.GuildID, row.ChannelID, row.UserID
}

func FormatName(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func ReplaceDefault(sessionID, name string) {
	if name == "" {
		return
	}
	row, _ := read(sessionID)
	if row.SessionID == "" {
		return
	}
	if row.Name != "" && !strings.HasPrefix(row.Name, "tg-") && !strings.HasPrefix(row.Name, "dc-") && !strings.HasPrefix(row.Name, "ln-") {
		return
	}

	row.Name = name
	if err := write(row); err != nil {
		slog.Warn("configBot ReplaceDefault",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
}

func NeedTitle(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	row, _ := read(sessionID)
	return row.Name == "" || row.Name == sessionID
}

func SetTitle(sessionID, title string) error {
	title = strings.TrimSpace(title)
	if title == "" || !NeedTitle(sessionID) {
		return nil
	}

	row, _ := read(sessionID)
	if row.SessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	row.Name = title
	return write(row)
}

func Save(sessionID, name, body string, force bool) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	if name == "" {
		name = sessionID
	}
	if body == "" {
		body = configs.DefaultSessionPrompt
	}

	row, exists := read(sessionID)
	if !force && exists {
		return nil
	}

	row.SessionID = sessionID
	row.Name = name
	row.Rule = body
	if err := write(row); err != nil {
		return err
	}

	dir := filesystem.SessionDir(sessionID)
	if err := go_pkg_filesystem.CheckDir(dir, true); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem CheckDir [%s]: %w", dir, err)
	}
	return nil
}
