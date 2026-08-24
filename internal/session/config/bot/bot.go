package configBot

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"unicode"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	DefaultModel     = "auto"
	DefaultReasoning = "medium"
)

var (
	frontmatterRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?(.*)$`)
	fieldRegex       = regexp.MustCompile(`(?m)^(\w+):\s*(.+)$`)
)

type Bot struct {
	Name      string `json:"name"`
	Model     string `json:"model,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	Body      string `json:"body"`
}

func read(sessionID string) Bot {
	bot, _ := readBot(sessionID)
	return bot
}

func readBot(sessionID string) (Bot, bool) {
	if sessionID == "" {
		return Bot{}, true
	}

	path := filesystem.BotPath(sessionID)
	if !go_pkg_filesystem_reader.Exists(path) {
		return migrate(sessionID), true
	}

	bot, err := go_pkg_filesystem.ReadJSON[Bot](path)
	if err == nil {
		return bot, true
	}

	broken := path + ".invalid"
	if renameErr := os.Rename(path, broken); renameErr != nil {
		slog.Warn("bot.json read",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return Bot{}, false
	}
	slog.Warn("bot.json unreadable, parked",
		slog.String("session", sessionID),
		slog.String("moved_to", broken),
		slog.String("error", err.Error()))
	return migrate(sessionID), true
}

func migrate(sessionID string) Bot {
	legacy := filesystem.LegacyBotPath(sessionID)
	if !go_pkg_filesystem_reader.Exists(legacy) {
		return Bot{}
	}

	data, err := go_pkg_filesystem.ReadText(legacy)
	if err != nil {
		return Bot{}
	}

	bot := Bot{Body: strings.TrimSpace(data)}
	if m := frontmatterRegex.FindStringSubmatch(data); len(m) >= 3 {
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
	}

	if err := writeBotFile(sessionID, bot); err != nil {
		slog.Warn("bot.md migrate",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
		return bot
	}
	if err := os.Remove(legacy); err != nil {
		slog.Warn("bot.md remove",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	return bot
}

func writeBotFile(sessionID string, bot Bot) error {
	if err := go_pkg_filesystem.WriteJSON(filesystem.BotPath(sessionID), bot, true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.WriteJSON: %w", err)
	}
	return nil
}

func Get(sessionID string) (name, body string) {
	bot := read(sessionID)
	return bot.Name, bot.Body
}

func GetModel(sessionID string) (model, reasoning string) {
	bot := read(sessionID)
	model = bot.Model
	reasoning = bot.Reasoning
	if model == "" {
		model = DefaultModel
	}
	if reasoning == "" {
		reasoning = DefaultReasoning
	}
	return model, reasoning
}

func SetModel(sessionID, model, reasoning string) {
	if sessionID == "" {
		return
	}
	bot, ok := readBot(sessionID)
	if !ok {
		return
	}
	if model != "" {
		bot.Model = model
	}
	if reasoning != "" {
		bot.Reasoning = reasoning
	}
	writeBotFile(sessionID, bot)
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
	if sessionID == "" || name == "" {
		return
	}
	bot, ok := readBot(sessionID)
	if !ok {
		return
	}
	if bot.Name != "" && !strings.HasPrefix(bot.Name, "tg-") && !strings.HasPrefix(bot.Name, "dc-") {
		return
	}
	bot.Name = name
	writeBotFile(sessionID, bot)
}

func NeedTitle(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	bot := read(sessionID)
	return bot.Name == "" || bot.Name == sessionID
}

func SetTitle(sessionID, title string) error {
	title = strings.TrimSpace(title)
	if title == "" || !NeedTitle(sessionID) {
		return nil
	}
	bot, ok := readBot(sessionID)
	if !ok {
		return fmt.Errorf("bot record for %s is unreadable; refusing to overwrite it", sessionID)
	}
	bot.Name = title
	return writeBotFile(sessionID, bot)
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

	current, ok := readBot(sessionID)
	if !ok {
		return fmt.Errorf("bot record for %s is unreadable; refusing to overwrite it", sessionID)
	}
	if !force && go_pkg_filesystem_reader.Exists(filesystem.BotPath(sessionID)) {
		return nil
	}

	return writeBotFile(sessionID, Bot{
		Name:      name,
		Model:     current.Model,
		Reasoning: current.Reasoning,
		Body:      body,
	})
}
