package history

import (
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/configs"
	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/store"
	provider "github.com/pardnchiu/go-llm-router/core"
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
		parts = append(parts, "sendAt: "+time.Unix(0, r.SendAt).Format(configs.TIME_LAYOUT))
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
		list[i].Content = configs.MESSAGE_PREFIX_REGEX.ReplaceAllString(str, "")
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
