package tui

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pardnchiu/go-llm-router/core"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	sessionTUI "github.com/pardnchiu/agenvoy/internal/session/tui"
)

var userWrapperRe = regexp.MustCompile(`^---\n當前時間:[^\n]*\n(?:[^\n]+\n)*?---\n`)
var cacheHitPctRe = regexp.MustCompile(`^\((\d+)%\)$`)

type parsedAction struct {
	timestamp string
	hash      string
	kind      string
	body      string
}

func cutBracket(s string) (inside, rest string, ok bool) {
	if !strings.HasPrefix(s, "[") {
		return "", "", false
	}
	inside, rest, ok = strings.Cut(s[1:], "]")
	return
}

func parseActionLine(raw string) (parsedAction, bool) {
	ts, rest, ok := cutBracket(raw)
	if !ok {
		return parsedAction{}, false
	}
	mid, rest, ok := cutBracket(rest)
	if !ok {
		return parsedAction{}, false
	}

	var hash, kind string
	if third, after, ok := cutBracket(rest); ok {
		hash = mid
		kind = third
		rest = after
	} else {
		hash = sessionTUI.Default
		kind = mid
	}

	return parsedAction{
		timestamp: ts,
		hash:      hash,
		kind:      kind,
		body:      strings.TrimSpace(rest),
	}, true
}

func renderActionLine(p parsedAction, width int) string {
	body := strings.ReplaceAll(p.body, sessionLog.ActionNewlineMarker, "\n")

	switch p.kind {
	case "user":
		body = userWrapperRe.ReplaceAllString(body, "")
		if strings.Contains(body, "[Resumed Task") {
			return ""
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return ""
		}
		return messageBlock(body)

	case "assistant":
		str := strings.TrimSpace(body)
		if str == "" {
			return ""
		}
		return renderEvent(agentTypes.Event{Type: agentTypes.EventText, Text: str}, width)

	case "tool_skipped":
		name, args, _ := strings.Cut(body, " ")
		return renderEvent(agentTypes.Event{
			Type:     agentTypes.EventToolSkipped,
			ToolName: name,
			ToolArgs: args,
		}, width)

	case "error":
		name, msg, _ := strings.Cut(body, " ")
		return renderEvent(agentTypes.Event{
			Type:     agentTypes.EventExecError,
			ToolName: name,
			Text:     msg,
			Err:      errors.New(msg),
		}, width)

	case "done":
		return renderEvent(formatDone(body), width, formatLogTimestamp(p.timestamp))

	case "canceled":
		event := formatDone(body)
		event.Type = agentTypes.EventCanceled
		return renderEvent(event, width, formatLogTimestamp(p.timestamp))

	case "skill_result":
		str := strings.TrimSpace(body)
		if str == "" {
			return ""
		}
		return renderEvent(agentTypes.Event{Type: agentTypes.EventSkillResult, Text: str}, width)
	}
	return ""
}

func formatLogTimestamp(ts string) string {
	if t, err := time.Parse("2006-01-02 15:04:05.000", ts); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return ts
}

func formatLog(raw string, width int) (kind, line string) {
	p, ok := parseActionLine(raw)
	if !ok {
		return "", ""
	}
	return p.kind, renderActionLine(p, width)
}

func renderEvent(ev agentTypes.Event, width int, finishedAt ...string) string {
	ts := ""
	if len(finishedAt) > 0 {
		ts = finishedAt[0]
	}
	line, ok := renderAgentEvent(nil, ev, "", "", width, ts)
	if !ok {
		return ""
	}
	return line
}

func formatDone(body string) agentTypes.Event {
	event := agentTypes.Event{Type: agentTypes.EventDone}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return event
	}
	if !strings.Contains(fields[0], "=") {
		event.Model = fields[0]
		fields = fields[1:]
	}

	var usage provider.Usage
	var hasUsage bool
	hitPct := -1
	for _, f := range fields {
		if m := cacheHitPctRe.FindStringSubmatch(f); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				hitPct = n
			}
			continue
		}
		k, v, found := strings.Cut(f, "=")
		if !found {
			continue
		}
		switch k {
		case "dur":
			if d, err := time.ParseDuration(v); err == nil {
				event.Duration = d
			}
		case "in":
			if n, err := strconv.Atoi(v); err == nil {
				usage.Input = n
				hasUsage = true
			}
		case "out":
			if n, err := strconv.Atoi(v); err == nil {
				usage.Output = n
				hasUsage = true
			}
		}
	}
	if hitPct >= 0 && usage.Input > 0 {
		usage.CacheRead = usage.Input * hitPct / 100
		usage.Input -= usage.CacheRead
	}
	if hasUsage {
		event.Usage = &usage
	}
	return event
}
