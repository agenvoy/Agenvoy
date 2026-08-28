package sessionDiscord

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func New(guildID, channelID, userID string) (string, error) {
	if guildID == "" {
		guildID = "dm"
	}
	if channelID == "" {
		channelID = "ch"
	}

	var key, boundGuild, boundUser string
	if guildID == "dm" {
		key = fmt.Sprintf("%s_%s", channelID, userID)
		boundUser = userID
	} else {
		key = fmt.Sprintf("%s_%s", guildID, channelID)
		boundGuild = guildID
	}
	sum := sha256.Sum256([]byte(key))
	sessionID := "dc-" + hex.EncodeToString(sum[:])

	botName := configBot.FormatName(utils.LookupChatName(filesystem.DiscordAuthPath, channelID))
	if err := configBot.Save(sessionID, botName, "", false); err != nil {
		slog.Debug("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if botName != "" {
		configBot.ReplaceDefault(sessionID, botName)
	}
	if err := configBot.SetChannel(sessionID, "", boundGuild, channelID, boundUser); err != nil {
		return "", fmt.Errorf("configBot.SetChannel: %w", err)
	}
	return sessionID, nil
}

func GetChannel(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is required")
	}

	_, _, channelID, _ := configBot.GetChannel(sessionID)
	return channelID, nil
}
