package compact

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentSummary "github.com/pardnchiu/agenvoy/internal/agents/exec/summary"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

type compactExchange struct {
	start int
	end   int
}

var (
	compactJSONRegex = regexp.MustCompile(`\{[^{}]*"remove"\s*:\s*\[[^]]*\][^{}]*\}`)
)

func SessionHistory(ctx context.Context, sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session id is required")
	}

	compactCtx, cancel := context.WithTimeout(
		agentTypes.WithSessionID(ctx, sessionID),
		2*time.Duration(filesystem.AgentSendTimeoutSec)*time.Second,
	)
	defer cancel()

	old, _ := sessionHistory.Get(sessionID)
	if len(old) < 4 {
		syncActionLog(sessionID, old)
		return 0, nil
	}

	agent := agents.Registry().Fallback
	if agent == nil {
		return 0, fmt.Errorf("no agent available for compact")
	}

	exchanges := groupExchanges(old)
	if len(exchanges) < 2 {
		syncActionLog(sessionID, old)
		return 0, nil
	}

	removeSet := preFilter(old, exchanges)

	var remaining []int
	for i := range exchanges {
		if !removeSet[i] {
			remaining = append(remaining, i)
		}
	}

	if len(remaining) >= 2 {
		llmRemove, err := identifyRemovable(compactCtx, agent, old, exchanges, remaining)
		if err != nil {
			slog.Warn("compact: LLM pass failed, using pre-filter only",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
		} else {
			for idx := range llmRemove {
				removeSet[idx] = true
			}
		}
	}

	if len(removeSet) == 0 {
		syncActionLog(sessionID, old)
		return 0, nil
	}

	kept := make([]sessionHistory.Record, 0, len(old))
	removed := 0
	for i, ex := range exchanges {
		if removeSet[i] {
			removed += ex.end - ex.start
			continue
		}
		kept = append(kept, old[ex.start:ex.end]...)
	}
	if removed == 0 {
		syncActionLog(sessionID, old)
		return 0, nil
	}

	if err := sessionHistory.Replace(sessionID, kept); err != nil {
		return 0, fmt.Errorf("history replace: %w", err)
	}

	syncActionLog(sessionID, kept)
	return removed, nil
}

func syncActionLog(sessionID string, messages []sessionHistory.Record) {
	var keptContents []string
	for _, msg := range messages {
		if msg.Role == "user" {
			keptContents = append(keptContents, msg.Text())
		}
	}
	sessionLog.RetainExchanges(sessionID, keptContents)
}

func groupExchanges(messages []sessionHistory.Record) []compactExchange {
	var list []compactExchange
	current := -1
	for i, msg := range messages {
		if msg.Role == "user" {
			if current >= 0 {
				list = append(list, compactExchange{start: current, end: i})
			}
			current = i
		}
	}
	if current >= 0 {
		list = append(list, compactExchange{start: current, end: len(messages)})
	}
	return list
}

func preFilter(messages []sessionHistory.Record, exchanges []compactExchange) map[int]bool {
	removeSet := make(map[int]bool)
	seen := make(map[string]bool)

	for i := len(exchanges) - 1; i >= 0; i-- {
		content := strings.TrimSpace(messages[exchanges[i].start].Text())
		if content == "" || seen[content] {
			removeSet[i] = true
			continue
		}
		seen[content] = true
	}

	return removeSet
}

func identifyRemovable(ctx context.Context, agent agentTypes.Agent, messages []sessionHistory.Record, exchanges []compactExchange, indices []int) (map[int]bool, error) {
	var sb strings.Builder
	for _, i := range indices {
		ex := exchanges[i]
		fmt.Fprintf(&sb, "=== Exchange %d ===\n", i)
		for _, msg := range messages[ex.start:ex.end] {
			fmt.Fprintf(&sb, "[%s] %s\n", msg.Role, go_pkg_utils.TruncateString(msg.Text(), 500))
		}
		sb.WriteString("\n")
	}

	str := agentSummary.Send(ctx, agent, agentTypes.SessionIDFrom(ctx), nil, []provider.Message{
		{Role: "system", Content: strings.TrimSpace(configs.CompactHistoryPrompt)},
		{Role: "user", Content: sb.String()},
	}, provider.ReasoningMedium, CheckThreshold)
	if str == "" {
		return nil, fmt.Errorf("no usable response")
	}

	return parseRemoveIndices(str, len(exchanges))
}

func parseRemoveIndices(raw string, exchangeCount int) (map[int]bool, error) {
	match := compactJSONRegex.FindString(raw)
	if match == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result struct {
		Remove []int `json:"remove"`
	}
	if err := json.Unmarshal([]byte(match), &result); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	dic := make(map[int]bool, len(result.Remove))
	for _, idx := range result.Remove {
		if idx < 0 || idx >= exchangeCount {
			slog.Warn("compact: out-of-range index ignored", slog.Int("index", idx))
			continue
		}
		dic[idx] = true
	}
	return dic, nil
}
