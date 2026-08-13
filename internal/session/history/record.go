package history

import (
	"regexp"
	"strings"
	"time"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	provider "github.com/pardnchiu/go-llm-router/core"
)

const TimeLayout = "2006-01-02 15:04:05"

var (
	prefixRegex       = regexp.MustCompile(`\A\s*(?:sendAt|sender|channelId)\s*:[^\n]*\n`)
	prefixHeadRegex   = regexp.MustCompile(`^(?:sendAt|sender|channelId)\s*:`)
	legacyBlockRegex  = regexp.MustCompile(`\A\s*-{3,}\n?(?:(?:當前時間|工作目錄|傳送者|當前 chat ID|當前 channel)[^\n]*\n?)+-{3,}\n?`)
	legacyLineRegex   = regexp.MustCompile(`\A\s*(?:當前時間|工作目錄|傳送者|當前 chat ID|當前 channel)\s*[:：][^\n]*\n?`)
	legacyTimeRegex   = regexp.MustCompile(`當前時間:\s*(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`)
	legacySenderRegex = regexp.MustCompile(`傳送者:\s*([^\n]*)`)
)

type Record struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
	SendAt  int64  `json:"sendAt,omitempty"`
	Sender  string `json:"sender,omitempty"`
}

func (r Record) Prefix() string {
	var parts []string
	if r.SendAt > 0 {
		parts = append(parts, "sendAt: "+time.Unix(0, r.SendAt).Format(TimeLayout))
	}
	if r.Sender != "" {
		parts = append(parts, "sender: "+r.Sender)
	}
	return strings.Join(parts, ", ")
}

func (r Record) Message() provider.Message {
	return provider.Message{
		Role:    r.Role,
		Content: WithPrefix(r.Prefix(), r.Content),
	}
}

func (r Record) Text() string {
	return historyStore.ExtractContent(r.Content)
}

func WithPrefix(prefix string, content any) any {
	if prefix == "" {
		return content
	}

	switch value := content.(type) {
	case string:
		return prefix + "\n" + value

	case []provider.ContentPart:
		for i, part := range value {
			if part.Type != "text" {
				continue
			}
			out := append([]provider.ContentPart(nil), value...)
			out[i].Text = prefix + "\n" + part.Text
			return out
		}
		return append([]provider.ContentPart{{Type: "text", Text: prefix}}, value...)

	case nil:
		return prefix

	default:
		return content
	}
}

func StripPrefix(content string) string {
	content = legacyBlockRegex.ReplaceAllString(content, "")
	for {
		trimmed := legacyLineRegex.ReplaceAllString(content, "")
		if trimmed == content {
			break
		}
		content = trimmed
	}
	return prefixRegex.ReplaceAllString(content, "")
}

func HasPrefix(line string) bool {
	return prefixHeadRegex.MatchString(line)
}

func Messages(list []Record) []provider.Message {
	if len(list) == 0 {
		return nil
	}
	out := make([]provider.Message, 0, len(list))
	for _, r := range list {
		out = append(out, r.Message())
	}
	return out
}

func normalize(list []Record) []Record {
	for i, r := range list {
		str, ok := r.Content.(string)
		if !ok {
			continue
		}
		if r.SendAt == 0 {
			if match := legacyTimeRegex.FindStringSubmatch(str); len(match) > 1 {
				if t, err := time.ParseInLocation(TimeLayout, match[1], time.Local); err == nil {
					list[i].SendAt = t.UnixNano()
				}
			}
		}
		if r.Sender == "" {
			if match := legacySenderRegex.FindStringSubmatch(str); len(match) > 1 {
				list[i].Sender = strings.TrimSpace(match[1])
			}
		}
		list[i].Content = StripPrefix(str)
	}
	return list
}

func rows(list []Record) []historyStore.Message {
	out := make([]historyStore.Message, 0, len(list))
	for _, r := range list {
		content := r.Text()
		if strings.TrimSpace(content) == "" {
			continue
		}
		out = append(out, historyStore.Message{
			SendAt:  r.SendAt,
			Role:    r.Role,
			Content: content,
			Sender:  r.Sender,
		})
	}
	return out
}
