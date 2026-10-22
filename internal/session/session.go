package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	configbot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type NamedSession struct {
	SessionID string
	SelfID    string
	Name      string
	Role      string
}

func New(prefix string) (string, error) {
	uuid := go_pkg_utils.UUID()
	if uuid == "" {
		return "", utilsmsg.New(utilsmsg.Internal, "go_pkg_utils.UUID", "no UUID generated")
	}

	sessionID := prefix + uuid
	if err := configbot.Save(sessionID, "", "", false); err != nil {
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
		selfID, name, body := configbot.GetPersona(sid)
		if selfID == "" {
			continue
		}
		list = append(list, NamedSession{
			SessionID: sid,
			SelfID:    selfID,
			Name:      name,
			Role:      strings.TrimSpace(body),
		})
	}
	return list
}

func GetSessionIDBySelfID(selfID string) string {
	if selfID == "" {
		return ""
	}
	sid := historystore.FindSessionBySelfID(context.Background(), selfID)
	if strings.HasPrefix(sid, "temp-") {
		return ""
	}
	return sid
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

	botName := configbot.FormatName(utils.LookupChatName(filesystem.LineAuthPath, target))
	if err := configbot.Save(sessionID, botName, "", false); err != nil {
		slog.Warn("configBot Save",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}
	if botName != "" {
		configbot.ReplaceDefault(sessionID, botName)
	}
	if err := configbot.SetChannel(sessionID, target, "", "", ""); err != nil {
		return "", utilsmsg.Trace("configbot.SetChannel", "", err)
	}
	return sessionID, nil
}

func GetLineTarget(sessionID string) (string, error) {
	if sessionID == "" {
		return "", utilsmsg.New(utilsmsg.Invalid, "sessionID", "is required")
	}

	chatID, _, _, _ := configbot.GetChannel(sessionID)
	return chatID, nil
}
