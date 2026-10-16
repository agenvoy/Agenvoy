package subagent

import (
	"strings"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func ensureSession(sessionID string) (string, error) {
	sessID := strings.TrimSpace(sessionID)
	if sessID != "" && go_pkg_filesystem_reader.IsDir(filesystem.SessionDir(sessID)) {
		return sessID, nil
	}

	// * find a idleID one
	if idleID := session.FindTemp(); idleID != "" {
		return idleID, nil
	}

	// * no idle exist, create new temp
	newID, err := session.New("temp-")
	if err != nil {
		return "", utilsmsg.Trace("create temp session", "", err)
	}
	return newID, nil
}
