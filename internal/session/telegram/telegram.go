package sessionTelegram

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func New(chatID int64) (string, error) {
	key := fmt.Sprintf("tg_%d", chatID)
	sum := sha256.Sum256([]byte(key))
	sessionID := "tg-" + hex.EncodeToString(sum[:])

	botName := configBot.FormatName(utils.LookupChatName(filesystem.TelegramAuthPath, strconv.FormatInt(chatID, 10)))
	if err := configBot.Save(sessionID, botName, "", false); err != nil {
		slog.Debug("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if botName != "" {
		configBot.ReplaceDefault(sessionID, botName)
	}
	if err := configBot.SetChannel(sessionID, strconv.FormatInt(chatID, 10), "", "", ""); err != nil {
		return "", fmt.Errorf("configBot.SetChannel: %w", err)
	}
	return sessionID, nil
}

func GetChat(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is required")
	}

	chatID, _, _, _ := configBot.GetChannel(sessionID)
	return chatID, nil
}
