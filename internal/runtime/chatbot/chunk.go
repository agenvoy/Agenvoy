package chatbot

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	telegramChunkBytes = 3200
	discordChunkBytes  = 1600
)

var (
	tagRegex   = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9-]*)([^>]*)>`)
	fenceRegex = regexp.MustCompile("```([a-zA-Z0-9_+-]*)")
)

type openTag struct {
	name  string
	attrs string
}

func Chunk(ch Channel, str string) []string {
	limit := telegramChunkBytes
	if ch == Discord {
		limit = discordChunkBytes
	}
	if len(str) <= limit {
		return []string{str}
	}

	var chunks []string
	var tagStack []openTag
	var fenceLang string
	var fenceOpen bool
	pos := 0

	for pos < len(str) {
		remaining := str[pos:]

		var prefix string
		switch ch {
		case Telegram:
			prefix = renderOpenTags(tagStack)
		case Discord:
			if fenceOpen {
				prefix = "```" + fenceLang + "\n"
			}
		}

		budget := limit - len(prefix)
		if budget <= 0 {
			budget = limit / 2
		}
		if len(remaining) <= budget {
			chunks = append(chunks, prefix+remaining)
			break
		}

		window := remaining[:budget]
		idx := strings.LastIndex(window, "\n\n")
		if idx < 0 {
			idx = strings.LastIndex(window, "\n")
		}
		if idx < 0 {
			idx = strings.LastIndex(window, " ")
		}
		if idx <= 0 {
			idx = budget
			for idx > 0 && !utf8.RuneStart(remaining[idx]) {
				idx--
			}
		}

		body := strings.TrimRight(remaining[:idx], " \n")
		full := prefix + body

		var suffix string
		switch ch {
		case Telegram:
			tagStack = scanOpenTags(full)
			suffix = renderCloseTags(tagStack)
		case Discord:
			fenceLang, fenceOpen = scanFenceState(full)
			if fenceOpen {
				suffix = "\n```"
			}
		}

		chunks = append(chunks, full+suffix)
		pos += idx
		for pos < len(str) && (str[pos] == '\n' || str[pos] == ' ') {
			pos++
		}
	}
	return chunks
}

func scanOpenTags(s string) []openTag {
	var stack []openTag
	for _, m := range tagRegex.FindAllStringSubmatch(s, -1) {
		closing := m[1] == "/"
		name := strings.ToLower(m[2])
		attrs := m[3]
		if isVoidTag(name) {
			continue
		}
		if !closing {
			stack = append(stack, openTag{name: name, attrs: attrs})
			continue
		}
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].name == name {
				stack = append(stack[:i], stack[i+1:]...)
				break
			}
		}
	}
	return stack
}

func renderOpenTags(stack []openTag) string {
	var sb strings.Builder
	for _, t := range stack {
		sb.WriteString("<")
		sb.WriteString(t.name)
		sb.WriteString(t.attrs)
		sb.WriteString(">")
	}
	return sb.String()
}

func renderCloseTags(stack []openTag) string {
	var sb strings.Builder
	for i := len(stack) - 1; i >= 0; i-- {
		sb.WriteString("</")
		sb.WriteString(stack[i].name)
		sb.WriteString(">")
	}
	return sb.String()
}

func isVoidTag(name string) bool {
	switch name {
	case "br", "hr", "img", "input", "meta", "link":
		return true
	}
	return false
}

func scanFenceState(s string) (string, bool) {
	matches := fenceRegex.FindAllStringSubmatch(s, -1)
	if len(matches)%2 == 0 {
		return "", false
	}
	return matches[len(matches)-1][1], true
}
