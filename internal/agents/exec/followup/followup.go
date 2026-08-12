package followup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const (
	Timeout      = 20 * time.Second
	maxTurns     = 6
	maxTurnRunes = 600
	maxTitle     = 40
	maxSuggest   = 3
	suggestRunes = 40
)

var jsonRegex = regexp.MustCompile(`(?s)\{.*\}`)

type Result struct {
	Title    string   `json:"title"`
	Suggests []string `json:"follow_ups"`
}

func (r Result) Empty() bool {
	return r.Title == "" && len(r.Suggests) == 0
}

func Generate(ctx context.Context, sessionID string, histories []sessionHistory.Record, needTitle bool) Result {
	transcript := transcript(histories)
	if transcript == "" {
		return Result{}
	}

	agent := pick()
	if agent == nil {
		return Result{}
	}

	sendCtx, cancel := context.WithTimeout(agentTypes.WithSessionID(ctx, sessionID), Timeout)
	defer cancel()

	resp, _, err := agent.Send(sendCtx, []provider.Message{
		{Role: "system", Content: prompt(needTitle)},
		{Role: "user", Content: "<conversation>\n" + transcript + "\n</conversation>"},
	}, nil, provider.ReasoningNone, fast.Mode())
	if err != nil {
		slog.Warn("followup.Generate",
			slog.String("session", sessionID),
			slog.String("model", agent.Name()),
			slog.String("error", err.Error()))
		return Result{}
	}
	if len(resp.Choices) == 0 {
		return Result{}
	}

	prov, model, _ := strings.Cut(agent.Name(), "@")
	usagelog.Append(sessionID, prov, model, resp.Usage)

	content, _ := resp.Choices[0].Message.Content.(string)
	return parse(content)
}

func prompt(needTitle bool) string {
	shape := `{"follow_ups": ["<2-8 words>", "<2-8 words>", "<2-8 words>"]}`
	rule := ""
	if needTitle {
		shape = `{"title": "<3-6 words>", "follow_ups": ["<2-8 words>", "<2-8 words>", "<2-8 words>"]}`
		rule = "- `title`: names what the whole conversation is about, not just the last turn.\n"
	}

	return strings.TrimSpace(strings.NewReplacer(
		"{{.Shape}}", shape,
		"{{.TitleRule}}", rule,
	).Replace(configs.FollowupPrompt))
}

func pick() agentTypes.Agent {
	if bot := agents.DispatcherBot(); bot != nil {
		return bot
	}
	if bot := agents.SummaryBot(); bot != nil {
		return bot
	}
	return agents.Registry().Fallback
}

func parse(content string) Result {
	match := jsonRegex.FindString(content)
	if match == "" {
		return Result{}
	}

	var raw Result
	if err := json.Unmarshal([]byte(match), &raw); err != nil {
		return Result{}
	}

	out := Result{Title: clamp(raw.Title, maxTitle)}
	seen := make(map[string]bool, len(raw.Suggests))
	for _, s := range raw.Suggests {
		s = clamp(s, suggestRunes)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out.Suggests = append(out.Suggests, s)
		if len(out.Suggests) == maxSuggest {
			break
		}
	}
	return out
}

func clamp(str string, limit int) string {
	str = strings.TrimSpace(strings.ReplaceAll(str, "\n", " "))
	if utf8.RuneCountInString(str) <= limit {
		return str
	}
	return string([]rune(str)[:limit])
}

func transcript(histories []sessionHistory.Record) string {
	if len(histories) > maxTurns {
		histories = histories[len(histories)-maxTurns:]
	}

	var sb strings.Builder
	for _, record := range histories {
		text := clamp(record.Text(), maxTurnRunes)
		if text == "" {
			continue
		}
		fmt.Fprintf(&sb, "[%s] %s\n", record.Role, text)
	}
	return strings.TrimSpace(sb.String())
}
