package sessionTelegram

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	configbot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func New(chatID int64) (string, error) {
	key := fmt.Sprintf("tg_%d", chatID)
	sum := sha256.Sum256([]byte(key))
	sessionID := "tg-" + hex.EncodeToString(sum[:])

	botName := configbot.FormatName(utils.LookupChatName(filesystem.TelegramAuthPath, strconv.FormatInt(chatID, 10)))
	if err := configbot.Save(sessionID, botName, "", false); err != nil {
		slog.Debug("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if botName != "" {
		configbot.ReplaceDefault(sessionID, botName)
	}
	if err := configbot.SetChannel(sessionID, strconv.FormatInt(chatID, 10), "", "", ""); err != nil {
		return "", utilsmsg.Trace("configbot.SetChannel", "", err)
	}
	return sessionID, nil
}

func GetChat(sessionID string) (string, error) {
	if sessionID == "" {
		return "", utilsmsg.New(utilsmsg.Invalid, "session id", "required")
	}

	chatID, _, _, _ := configbot.GetChannel(sessionID)
	return chatID, nil
}
