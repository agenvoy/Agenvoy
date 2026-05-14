package exec

import (
	"context"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

type DiscordPayload struct {
	SessionID string
	Text      string
	Model     string
	Usage     *agentTypes.Usage
	Prefix    string
}

type DiscordPushFunc func(ctx context.Context, payload DiscordPayload)

var ResultHook DiscordPushFunc

type DiscordPushKey struct{}

func SuppressDcPush(ctx context.Context) context.Context {
	return context.WithValue(ctx, DiscordPushKey{}, true)
}

func isDcPushSuppressed(ctx context.Context) bool {
	v, _ := ctx.Value(DiscordPushKey{}).(bool)
	return v
}

type DiscordPushPrefix struct{}

func WithDcPushPrefix(ctx context.Context, prefix string) context.Context {
	return context.WithValue(ctx, DiscordPushPrefix{}, prefix)
}

func dcPushPrefix(ctx context.Context) string {
	v, _ := ctx.Value(DiscordPushPrefix{}).(string)
	return v
}
