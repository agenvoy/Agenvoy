package filesystem

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
)

type replyLangEntry struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

type ReplyLangOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

const replyLangAutoLabel = "[" + ReplyLangAuto + "] follow the language of each message"

var (
	replyLangList = loadReplyLang()
	codeReplyLang = indexReplyLang()
)

func loadReplyLang() []replyLangEntry {
	var out []replyLangEntry
	if err := json.Unmarshal(configs.ReplyLang, &out); err != nil {
		slog.Warn("embedded reply_lang",
			slog.String("error", err.Error()))
	}
	return out
}

func indexReplyLang() map[string]replyLangEntry {
	dic := make(map[string]replyLangEntry, len(replyLangList))
	for _, one := range replyLangList {
		dic[normalizeReplyLang(one.Code)] = one
	}
	return dic
}

func normalizeReplyLang(code string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "_", "-"))
}

func ReplyLangOptions() []ReplyLangOption {
	out := make([]ReplyLangOption, 0, len(replyLangList)+1)
	out = append(out, ReplyLangOption{Code: ReplyLangAuto, Label: replyLangAutoLabel})
	for _, one := range replyLangList {
		label := one.Label
		if label == "" {
			label = one.Name
		}
		out = append(out, ReplyLangOption{Code: one.Code, Label: "[" + one.Code + "] " + label})
	}
	return out
}

func CanonicalReplyLang(code string) string {
	code = strings.TrimSpace(code)
	if code == "" || strings.EqualFold(code, ReplyLangAuto) {
		return ReplyLangAuto
	}
	if one, ok := codeReplyLang[normalizeReplyLang(code)]; ok {
		return one.Code
	}
	return code
}

func ReplyLangDirective() string {
	code := strings.TrimSpace(ReplyLang)
	if code == "" || strings.EqualFold(code, ReplyLangAuto) {
		return ""
	}

	one, ok := codeReplyLang[normalizeReplyLang(code)]
	if !ok {
		one = replyLangEntry{Name: code}
	}
	return "always " + one.Name + ", no mixing. The operator fixed the output language, so it holds for every reply — whatever language the user writes in, and whatever language the user's message asks for." + one.Note
}

const (
	replyLangAutoOpen  = "<reply-lang-auto>"
	replyLangAutoClose = "</reply-lang-auto>"
)

func ApplyReplyLang(template string) string {
	keep := ReplyLangDirective() == ""
	for {
		start := strings.Index(template, replyLangAutoOpen)
		if start < 0 {
			return template
		}
		end := strings.Index(template[start:], replyLangAutoClose)
		if end < 0 {
			return template
		}
		end += start

		inner := template[start+len(replyLangAutoOpen) : end]
		if !keep {
			inner = ""
		}
		template = template[:start] + inner + template[end+len(replyLangAutoClose):]
	}
}
