package history

import (
	"strings"
	"time"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"

	"github.com/pardnchiu/agenvoy/configs"
	historystore "github.com/pardnchiu/agenvoy/internal/runtime/store"
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

func (r Record) Message() go_llm_router.Message {
	return go_llm_router.Message{
		Role:    r.Role,
		Content: WithPrefix(r.Prefix(), r.Content),
	}
}

func (r Record) Text() string {
	return historystore.ExtractContent(r.Content)
}

func WithPrefix(prefix string, content any) any {
	if prefix == "" {
		return content
	}

	switch value := content.(type) {
	case string:
		return prefix + "\n" + value

	case []go_llm_router.ContentPart:
		for i, part := range value {
			if part.Type != "text" {
				continue
			}
			out := append([]go_llm_router.ContentPart(nil), value...)
			out[i].Text = prefix + "\n" + part.Text
			return out
		}
		return append([]go_llm_router.ContentPart{{Type: "text", Text: prefix}}, value...)

	case nil:
		return prefix

	default:
		return content
	}
}

func Messages(list []Record) []go_llm_router.Message {
	if len(list) == 0 {
		return nil
	}
	out := make([]go_llm_router.Message, 0, len(list))
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

func rows(list []Record) []historystore.Message {
	out := make([]historystore.Message, 0, len(list))
	for _, r := range list {
		content := r.Text()
		if strings.TrimSpace(content) == "" {
			continue
		}
		out = append(out, historystore.Message{
			SendAt:  r.SendAt,
			Role:    r.Role,
			Content: content,
			Sender:  r.Sender,
		})
	}
	return out
}
