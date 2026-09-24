package line

import (
	"context"
	"fmt"
	"strings"

	go_bot_line "github.com/pardnchiu/go-bot/core/line"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
)

func getSession(ctx context.Context, in go_bot_line.Input, content string, data exec.ExecuteMeta) (*agentTypes.AgentSession, error) {
	sessionID, err := sessionManager.GetLineSession(in.UserID, in.GroupID, in.RoomID)
	if err != nil {
		return nil, fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetLineSession: %w", err)
	}

	data.SessionID = sessionID
	data.Content = content
	if sender := strings.TrimSpace(in.Username); sender != "" {
		data.Sender = sender
	} else {
		data.Sender = strings.TrimSpace(data.Sender)
	}

	sess, err := exec.GetSession(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("exec.GetSession: %w", err)
	}

	userText := strings.TrimSpace(data.Input)
	if userText == "" {
		userText = strings.TrimSpace(content)
	}
	sessionLog.Append(sessionID, userText)

	return sess, nil
}
