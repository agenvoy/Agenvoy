package exec

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

var (
	summaryLeakMarkerRegex = regexp.MustCompile(`(?i)(?:Prior Conversation Context|Prior summary|"key_decisions"\s*:\s*\[|"current_discussion"\s*:\s*\{)`)
	thinkTagRegex          = regexp.MustCompile(`(?is)<think>(.*?)</think>\s*`)
	thinkOpenRegex         = regexp.MustCompile(`(?i)<think>`)
	thinkCloseRegex        = regexp.MustCompile(`(?i)</think>`)
)

func Response(str string) string {
	// * remove system prefix
	str = configs.MESSAGE_PREFIX_REGEX.ReplaceAllString(str, "")
	if loc := summaryLeakMarkerRegex.FindStringIndex(str); loc != nil {
		dropped := []rune(strings.TrimSpace(str[loc[0]:]))
		head := dropped[:min(len(dropped), configs.LOG_HEAD_RUNES)]
		str = strings.TrimRight(str[:loc[0]], " \t\n\r#")
		slog.Debug("StripModelResponse summary leak stripped",
			slog.Int("dropped_chars", len(dropped)),
			slog.String("dropped_head", string(head)))
	}
	return strings.TrimSpace(str)
}

func splitThinkTag(s string) (think, rest string) {
	var parts []string
	for _, m := range thinkTagRegex.FindAllStringSubmatch(s, -1) {
		if t := strings.TrimSpace(m[1]); t != "" {
			parts = append(parts, t)
		}
	}
	rest = thinkTagRegex.ReplaceAllString(s, "")
	if loc := thinkOpenRegex.FindStringIndex(rest); loc != nil {
		if t := strings.TrimSpace(rest[loc[1]:]); t != "" {
			parts = append(parts, t)
		}
		rest = rest[:loc[0]]
	}
	rest = strings.TrimSpace(rest)
	if len(parts) == 0 {
		return "", rest
	}
	return strings.Join(parts, "\n"), rest
}

func guardrailLabel(content string) string {
	_, rest, ok := strings.Cut(content, configs.BAN_TAG)
	if !ok {
		return ""
	}
	label := strings.TrimSpace(rest)
	if cut := strings.IndexAny(label, " \t\n\r"); cut > 0 {
		label = label[:cut]
	}
	return strings.Trim(label, "[](){}:,.\"'`")
}

func guardrailRefusal(sessionID, model, content string) string {
	label := guardrailLabel(content)
	runes := []rune(content)
	head := runes[:min(len(runes), configs.LOG_HEAD_RUNES)]
	slog.Debug("guardrail refusal",
		slog.String("session", sessionID),
		slog.String("model", model),
		slog.String("label", label),
		slog.String("head", string(head)))

	refusal := filesystem.RefusalMessage()
	if label == "" {
		return refusal
	}
	return refusal + " (" + label + ")"
}
