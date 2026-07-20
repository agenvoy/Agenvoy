package agentTypes

import (
	"context"

	"github.com/pardnchiu/go-llm-router/core"
)

type Agent = provider.Agent

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
	SystemPrompts   []provider.Message
	OldHistories    []provider.Message
	SummaryMessage  provider.Message
	UserInput       provider.Message
	ToolHistories   []provider.Message
	Tools           []provider.Message
	Histories       []provider.Message
	BaseLen         int
	Stateless       bool
	ToolCheckpoint  int
}
