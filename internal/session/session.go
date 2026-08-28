package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type NamedSession struct {
	SessionID string
	Name      string
	Role      string
}

func New(prefix string) (string, error) {
	uuid := go_pkg_utils.UUID()
	if uuid == "" {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/utils UUID: no UUID generated")
	}

	sessionID := prefix + uuid
	if err := configBot.Save(sessionID, "", "", false); err != nil {
		slog.Debug("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	return sessionID, nil
}

func FindTemp() string {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return ""
	}
	for _, dir := range dirs {
		sid := dir.Name
		if !strings.HasPrefix(sid, "temp-") {
			continue
		}
		if ClaimIdle(sid) {
			return sid
		}
	}
	return ""
}

func ListSessions() []NamedSession {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return nil
	}
	var list []NamedSession
	for _, dir := range dirs {
		sid := dir.Name
		if strings.HasPrefix(sid, "temp-") {
			continue
		}
		name, body := configBot.Get(sid)
		if name == "" {
			continue
		}
		list = append(list, NamedSession{
			SessionID: sid,
			Name:      name,
			Role:      strings.TrimSpace(body),
		})
	}
	return list
}

func GetSessionID(name string) string {
	if name == "" {
		return ""
	}

	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return ""
	}

	for _, dir := range dirs {
		sid := dir.Name
		if strings.HasPrefix(sid, "temp-") {
			continue
		}

		botName, _ := configBot.Get(sid)
		if botName == "" {
			continue
		}
		if botName == name {
			return sid
		}
	}
	return ""
}

func GetLineSession(userID, groupID, roomID string) (string, error) {
	var key, target string
	switch {
	case groupID != "":
		key = "ln_g_" + groupID
		target = groupID
	case roomID != "":
		key = "ln_r_" + roomID
		target = roomID
	default:
		key = "ln_u_" + userID
		target = userID
	}
	sum := sha256.Sum256([]byte(key))
	sessionID := "ln-" + hex.EncodeToString(sum[:])

	botName := configBot.FormatName(utils.LookupChatName(filesystem.LineAuthPath, target))
	if err := configBot.Save(sessionID, botName, "", false); err != nil {
		slog.Warn("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if botName != "" {
		configBot.ReplaceDefault(sessionID, botName)
	}
	if err := configBot.SetChannel(sessionID, target, "", "", ""); err != nil {
		return "", fmt.Errorf("configBot.SetChannel: %w", err)
	}
	return sessionID, nil
}

func GetLineTarget(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is required")
	}

	chatID, _, _, _ := configBot.GetChannel(sessionID)
	return chatID, nil
}
