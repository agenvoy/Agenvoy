package claudeCode

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
)

const Provider = "claude-code"

func CheckBinary() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found; install Claude Code and run `claude` once to log in: %w", err)
	}
	return nil
}

func Is(name string) bool {
	prov, _, _ := strings.Cut(name, "@")
	return prov == Provider
}

var EnableClaudeCode bool

func Enabled() bool {
	return EnableClaudeCode && CheckBinary() == nil
}

type Agent struct {
	name  string
	model string
}

func New(name string) (*Agent, error) {
	prov, model, ok := strings.Cut(name, "@")
	if !ok || prov != Provider || model == "" {
		return nil, fmt.Errorf("invalid %s model name %q", Provider, name)
	}
	if err := CheckBinary(); err != nil {
		return nil, err
	}
	if !EnableClaudeCode {
		return nil, fmt.Errorf("%s is not enabled; run `agen stop`, then start with `agen --enable-claude-code`", Provider)
	}
	return &Agent{name: name, model: model}, nil
}

func (a *Agent) Name() string {
	return a.name
}

func (a *Agent) Send(ctx context.Context, messages []provider.Message, toolDefs []provider.Tool, reasoning provider.Reasoning, _ provider.Mode) (*provider.Output, int, error) {
	effort := effortOf(reasoning)
	system, rest := splitSystem(messages)
	sessionID := agentTypes.SessionIDFrom(ctx)

	if len(toolDefs) == 0 || sessionID == "" {
		p := &process{}
		if err := p.start(a.model, effort, len(toolDefs) > 0); err != nil {
			return nil, 0, err
		}
		defer p.stop()
		return p.turn(ctx, renderInitial(system, toolDefs, rest))
	}

	p := acquire(sessionID + "|" + a.name)
	defer p.mu.Unlock()

	spec := specOf(system, effort)
	tools := renderTools(toolDefs)
	list := fingerprints(rest)

	reusable := p.alive() && p.spec == spec
	var content []map[string]any
	if reusable && len(list) > len(p.sent) && slices.Equal(list[:len(p.sent)], p.sent) {
		content = renderMessages(rest[len(p.sent):], toolNames(rest))
	} else if i := answerIndex(rest, p.lastAnswer); reusable && i >= 0 {
		content = renderMessages(rest[i+1:], toolNames(rest))
	} else {
		p.stop()
		if err := p.start(a.model, effort, true); err != nil {
			return nil, 0, err
		}
		p.spec = spec
		p.tools = tools
		p.sent = nil
		content = renderInitial(system, toolDefs, rest)
	}
	if tools != p.tools {
		content = append([]map[string]any{{"type": "text", "text": tools + "\n\n"}}, content...)
		p.tools = tools
	}

	out, code, err := p.turn(ctx, content)
	if err != nil {
		p.stop()
		return nil, code, err
	}
	message := out.Choices[0].Message
	p.sent = append(list, fingerprint(message))
	p.lastAnswer = ""
	if len(message.ToolCalls) == 0 {
		p.lastAnswer = answerText(message.Content)
	}
	p.lastUse = time.Now()
	return out, code, nil
}

func answerIndex(messages []provider.Message, answer string) int {
	if answer == "" {
		return -1
	}
	for i, m := range slices.Backward(messages) {
		if m.Role != "assistant" || len(m.ToolCalls) > 0 {
			continue
		}
		if i == len(messages)-1 || answerText(m.Content) != answer {
			return -1
		}
		return i
	}
	return -1
}

func answerText(content any) string {
	return strings.TrimSpace(sessionHistory.StripPrefix(contentText(content)))
}

func effortOf(reasoning provider.Reasoning) string {
	switch reasoning {
	case provider.ReasoningMedium:
		return "medium"
	case provider.ReasoningHigh:
		return "high"
	case provider.ReasoningXHigh:
		return "xhigh"
	case provider.ReasoningMax:
		return "max"
	}
	return "low"
}
