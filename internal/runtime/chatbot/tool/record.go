package tool

import (
	"log/slog"
	"strings"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
)

const outboundModel = "forwarded"

func resolveChatbotSession(prefix, field, id string) string {
	dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir)
	if err != nil {
		return ""
	}

	for _, dir := range dirs {
		if !strings.HasPrefix(dir.Name, prefix) {
			continue
		}
		config, err := go_pkg_filesystem.ReadJSON[map[string]string](filesystem.SessionConfigPath(dir.Name))
		if err != nil {
			continue
		}
		if config[field] == id {
			return dir.Name
		}
	}
	return ""
}

func recordOutbound(sessionID, message string) {
	if sessionID == "" || message == "" {
		return
	}

	if err := sessionHistory.Append(sessionID, []sessionHistory.Record{{
		Role:    "assistant",
		Content: message,
		SendAt:  time.Now().UnixNano(),
	}}); err != nil {
		slog.Warn("sessionHistory.Append (push)",
			slog.String("session", sessionID),
			slog.String("error", err.Error()))
	}

	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventText, Text: message})
	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventTextDone})
	sessionLog.Record(sessionID, agentTypes.Event{Type: agentTypes.EventDone, Model: outboundModel})
}
