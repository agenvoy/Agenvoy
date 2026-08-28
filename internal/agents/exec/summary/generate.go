package agentSummary

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const (
	summaryChunkRunes = 32_000
)

var (
	fencedBlockRegex    = regexp.MustCompile("(?s)" + "```" + `(?:json|summary)\s*\n([\s\S]*?)\s*\n` + "```")
	summaryTagRegex     = regexp.MustCompile(`(?s)<summary>\s*([\s\S]*?)\s*</summary>`)
	summaryBracketRegex = regexp.MustCompile(`(?s)\[summary\]\s*([\s\S]*?)\s*\[/summary\]`)
)

func Generate(ctx context.Context, sessionID string, histories []sessionHistory.Record) error {
	// * caller supplies no agent, Send layers the summary model preference on top
	agent := agents.Registry().Fallback
	if agent == nil {
		return fmt.Errorf("no agent available")
	}

	summaryCtx := agentTypes.WithSessionID(ctx, sessionID)
	raw, dic := summary.Ensure(sessionID)
	cursor := summary.GetCursor(sessionID)

	// * step1: check summaried or not
	if cursor == "" && len(dic) > 0 {
		latest := latestTime(histories)
		summary.SaveCursor(sessionID, latest)
		summary.Save(sessionID, dic)
		return nil
	}

	// * step2: split chunks
	chunks := chunkHistories(histories, cursor)

	// * step3: start to summary
	oldSummary := string(raw)
	if oldSummary == "" {
		oldSummary = "{}"
	}

	for i, chunk := range chunks {
		newDic := generateSmmary(summaryCtx, agent, oldSummary, chunk)
		if newDic == nil {
			return fmt.Errorf("generateSmmary: returned nil at chunk %d/%d", i+1, len(chunks))
		}

		if raw, err := json.Marshal(newDic); err == nil {
			oldSummary = string(raw)
		}
		summary.Save(sessionID, newDic)

		if chunkLatest := latestTime(chunk); chunkLatest > cursor {
			cursor = chunkLatest
		}
		summary.SaveCursor(sessionID, cursor)
	}
	return nil
}

func chunkHistories(histories []sessionHistory.Record, cursor string) [][]sessionHistory.Record {
	if cursor != "" {
		latest := make([]sessionHistory.Record, 0, len(histories))
		for _, message := range histories {
			t := extractTime(message)
			if t == "" || t > cursor {
				latest = append(latest, message)
			}
		}
		histories = latest
	}
	if len(histories) == 0 {
		return nil
	}

	var list [][]sessionHistory.Record
	for i := 0; i < len(histories); {
		end, total := i, 0
		for end < len(histories) {
			n := contentRunes(histories[end])
			if end > i && total+n > summaryChunkRunes {
				break
			}
			total += n
			end++
		}
		if end < len(histories) && histories[end-1].Role == "user" {
			end++
		}
		list = append(list, histories[i:end])
		i = end
	}
	return list
}

func contentRunes(msg sessionHistory.Record) int {
	return utf8.RuneCountInString(msg.Text())
}

func generateSmmary(ctx context.Context, agent agentTypes.Agent, oldSummary string, histories []sessionHistory.Record) map[string]any {
	prompt := strings.NewReplacer(
		"{{.Summary}}", oldSummary,
	).Replace(strings.TrimSpace(configs.SummaryPrompt))

	var sb strings.Builder
	for _, hist := range histories {
		content := strings.TrimSpace(hist.Text())
		if content == "" {
			continue
		}
		fmt.Fprintf(&sb, "[%s]\n%s\n\n", hist.Role, content)
	}

	raw := fmt.Sprintf(`<conversation>
%s
</conversation>

The block above is the new conversation data since the previous summary.
Merge it into the previous summary per the rules above and return exactly one <summary> block.`, strings.TrimSpace(sb.String()))
	messages := []provider.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: raw},
	}

	result := Send(ctx, agent, agentTypes.SessionIDFrom(ctx), nil, messages, provider.ReasoningMedium, nil)
	if result == "" {
		return nil
	}

	var dic map[string]any
	for _, regex := range []*regexp.Regexp{fencedBlockRegex, summaryTagRegex, summaryBracketRegex} {
		if match := regex.FindStringSubmatchIndex(result); match != nil {
			part := result[match[2]:match[3]]
			if json.Unmarshal([]byte(part), &dic) == nil && summary.IsValid(dic) {
				return dic
			}
		}
	}

	if start := strings.Index(result, "{"); start >= 0 {
		var dic map[string]any
		if json.Unmarshal([]byte(result[start:]), &dic) == nil && summary.IsValid(dic) {
			return dic
		}
	}

	slog.Debug("generateSmmary: unparseable",
		slog.String("session", agentTypes.SessionIDFrom(ctx)),
		slog.String("preview", go_pkg_utils.TruncateString(result, 256)))
	return nil
}

func latestTime(messages []sessionHistory.Record) string {
	var str string
	for _, message := range messages {
		t := extractTime(message)
		if t > str {
			str = t
		}
	}
	return str
}

func extractTime(msg sessionHistory.Record) string {
	if msg.SendAt <= 0 {
		return ""
	}
	return time.Unix(0, msg.SendAt).Format(sessionHistory.TimeLayout)
}
