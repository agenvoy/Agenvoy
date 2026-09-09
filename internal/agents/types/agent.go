package agentTypes

import (
	"context"
	"encoding/json"

	provider "github.com/pardnchiu/go-llm-router/core"
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

type originCtxKey struct{}

func WithOrigin(ctx context.Context, origin string) context.Context {
	return context.WithValue(ctx, originCtxKey{}, origin)
}

func OriginFrom(ctx context.Context) string {
	origin, _ := ctx.Value(originCtxKey{}).(string)
	return origin
}

type deliverToCtxKey struct{}

func WithDeliverTo(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, deliverToCtxKey{}, sessionID)
}

func DeliverToFrom(ctx context.Context) string {
	sid, _ := ctx.Value(deliverToCtxKey{}).(string)
	return sid
}

type AgentRegistry struct {
	Registry map[string]Agent
	Entries  []AgentEntry
	Fallback Agent
}

type AgentEntry struct {
	Name string `json:"name"`
}

func (a *AgentEntry) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		a.Name = name
		return nil
	}
	var legacy struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return err
	}
	a.Name = legacy.Name
	return nil
}

type AgentSession struct {
	ID             string
	SystemPrompts  []provider.Message
	OldHistories   []provider.Message
	SummaryMessage provider.Message
	UserInput      provider.Message
	ToolHistories  []provider.Message
	Tools          []provider.Message
	Histories      []provider.Message
	BaseLen        int
	Sender         string
	UserSendAt     int64
	Stateless      bool
	ToolCheckpoint int
}
