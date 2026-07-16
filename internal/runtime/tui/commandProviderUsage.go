package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"
)

type ProviderUsageResult struct {
	lines []string
}

func (t TUI) commandProviderUsage() (TUI, tea.Cmd, bool) {
	hasCodex := false
	hasGrokOauth := false
	hasCopilot := false
	hasDeepseek := false
	hasOpenRouter := false
	for _, e := range exec.GetAgent() {
		prov, _, _ := strings.Cut(e.Name, "@")
		switch prov {
		case "codex":
			hasCodex = true
		case "grok-oauth":
			hasGrokOauth = true
		case "copilot":
			hasCopilot = true
		case "deepseek":
			hasDeepseek = true
		case "openrouter":
			hasOpenRouter = true
		}
	}
	if !hasCodex && !hasGrokOauth && !hasCopilot && !hasDeepseek && !hasOpenRouter {
		return t, nil, true
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		lines := make([]string, 5)
		var wg sync.WaitGroup

		fetch := func(idx int, run func() string) {
			defer wg.Done()
			lines[idx] = run()
		}

		if hasCodex {
			wg.Add(1)
			go fetch(0, func() string { return fetchProviderUsage(ctx, "Codex", "codex", openaicodex.Usage) })
		}
		if hasGrokOauth {
			wg.Add(1)
			go fetch(1, func() string { return fetchProviderUsage(ctx, "Grok", "grok-oauth", grokoauth.Usage) })
		}
		if hasCopilot {
			wg.Add(1)
			go fetch(2, func() string { return fetchProviderUsage(ctx, "Copilot", "copilot", copilot.Usage) })
		}
		if hasOpenRouter {
			wg.Add(1)
			go fetch(3, func() string { return fetchProviderBalance(ctx, "OpenRouter", "openrouter", openrouter.Usage) })
		}
		if hasDeepseek {
			wg.Add(1)
			go fetch(4, func() string { return fetchProviderBalance(ctx, "DeepSeek", "deepseek", deepseek.Usage) })
		}
		wg.Wait()

		result := make([]string, 0, len(lines))
		for _, line := range lines {
			if line != "" {
				result = append(result, line)
			}
		}
		send(ProviderUsageResult{lines: result})
	}()

	return t, nil, true
}

func fetchProviderUsage(ctx context.Context, label, prov string, fn func(context.Context, provider.Config) (float64, error)) string {
	cfg, err := agentKeychain.Config(ctx, prov)
	if err == nil {
		var remaining float64
		remaining, err = fn(ctx, cfg)
		if err == nil {
			return textStyle.Render(label+": ") + remainingPctStyle(remaining).Render(fmt.Sprintf("%.0f%%", remaining))
		}
	}
	return textStyle.Render(label+": ") + errorStyle.Render("failed")
}

func fetchProviderBalance(ctx context.Context, label, prov string, fn func(context.Context, provider.Config) (float64, error)) string {
	cfg, err := agentKeychain.Config(ctx, prov)
	if err == nil {
		var balance float64
		balance, err = fn(ctx, cfg)
		if err == nil {
			style := okayStyle
			if balance <= 0 {
				style = errorStyle
			}
			return textStyle.Render(label+": ") + style.Render(fmt.Sprintf("$%.2f", balance))
		}
	}
	return textStyle.Render(label+": ") + errorStyle.Render("failed")
}

func remainingPctStyle(remaining float64) lipgloss.Style {
	switch {
	case remaining < 20:
		return errorStyle
	case remaining < 50:
		return skillStyle
	default:
		return okayStyle
	}
}
