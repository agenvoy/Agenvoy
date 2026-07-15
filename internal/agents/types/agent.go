package agentTypes

import (
	"context"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

type Agent interface {
	Name() string
	Send(ctx context.Context, messages []Message, toolDefs []toolTypes.Tool) (*Output, error)
}

type sessionIDCtxKey struct{}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDCtxKey{}, sessionID)
}

func SessionIDFrom(ctx context.Context) string {
	sid, _ := ctx.Value(sessionIDCtxKey{}).(string)
	return sid
}

type AgentRegistry struct {
	Registry map[string]Agent
	Entries  []AgentEntry
	Fallback Agent
}

type AgentEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AgentSession struct {
	ID              string
	SystemPrompts   []Message
	OldHistories    []Message
	SummaryMessage  Message
	UserInput       Message
	ToolHistories   []Message
	Tools           []Message
	Histories       []Message
	BaseLen         int
	Stateless       bool
	VerifyRounds    int
	VerifyFeedbacks []string
	ToolCheckpoint  int
}
