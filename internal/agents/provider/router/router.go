package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/provider/claude"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/cloudflare"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/compat"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/copilot"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/deepseek"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/gemini"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/grok"
	grokoauth "github.com/pardnchiu/agenvoy/internal/agents/provider/grokOauth"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/nvidia"
	openrouter "github.com/pardnchiu/agenvoy/internal/agents/provider/openRouter"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/openai"
	openaicodex "github.com/pardnchiu/agenvoy/internal/agents/provider/openaiCodex"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var newFn = map[string]func(string) (agentTypes.Agent, error){
	"claude":     func(m string) (agentTypes.Agent, error) { return claude.New(m) },
	"openai":     func(m string) (agentTypes.Agent, error) { return openai.New(m) },
	"codex":      func(m string) (agentTypes.Agent, error) { return openaicodex.New(m) },
	"gemini":     func(m string) (agentTypes.Agent, error) { return gemini.New(m) },
	"grok":       func(m string) (agentTypes.Agent, error) { return grok.New(m) },
	"grok-oauth": func(m string) (agentTypes.Agent, error) { return grokoauth.New(m) },
	"copilot":    func(m string) (agentTypes.Agent, error) { return copilot.New(m) },
	"nvidia":     func(m string) (agentTypes.Agent, error) { return nvidia.New(m) },
	"cloudflare": func(m string) (agentTypes.Agent, error) { return cloudflare.New(m) },
	"deepseek":   func(m string) (agentTypes.Agent, error) { return deepseek.New(m) },
	"openrouter": func(m string) (agentTypes.Agent, error) { return openrouter.New(m) },
	"compat":     func(m string) (agentTypes.Agent, error) { return compat.New(m) },
}

func New(name string) (agentTypes.Agent, error) {
	providerFull, _, _ := strings.Cut(name, "@")
	prov, _, _ := strings.Cut(providerFull, "[")
	fn, ok := newFn[prov]
	if !ok {
		return nil, fmt.Errorf("router.New: unknown provider %q in %q", prov, name)
	}
	return fn(name)
}

func Send(ctx context.Context, name string, messages []agentTypes.Message, tools []toolTypes.Tool) (*agentTypes.Output, error) {
	agent, err := New(name)
	if err != nil {
		return nil, err
	}
	return agent.Send(ctx, messages, tools)
}
