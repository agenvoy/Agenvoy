package exec

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
	internalUtils "github.com/pardnchiu/agenvoy/internal/utils"
)

var (
	timestampHeaderRegex   = regexp.MustCompile(`\A\s*-{3,}\n(?:[^\n]*\n)*?-{3,}\n`)
	contextMetaLineRegex   = regexp.MustCompile(`\A\s*(?:當前時間|工作目錄|傳送者|當前 chat ID|當前 channel)\s*[:：][^\n]*\n?`)
	summaryBlockRegex      = regexp.MustCompile(`(?s)<summary>\s*[\s\S]*?\s*</summary>|\[summary\]\s*[\s\S]*?\s*\[/summary\]`)
	summaryLeakMarkerRegex = regexp.MustCompile(`(?i)(?:Prior Conversation Context|Prior summary|background summary of prior discussion|Strict rules:|"key_decisions"\s*:\s*\[|"current_discussion"\s*:\s*\{)`)
	thinkTagRegex          = regexp.MustCompile(`(?is)<think>(.*?)</think>\s*`)
	thinkOpenRegex         = regexp.MustCompile(`(?i)<think>`)
	thinkCloseRegex        = regexp.MustCompile(`(?i)</think>`)
)

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

func isGuardrailRefusal(content string) bool {
	return strings.Contains(content, configs.GuardrailSentinel)
}

func StripModelResponse(str string) string {
	str = timestampHeaderRegex.ReplaceAllString(str, "")
	for {
		trimmed := contextMetaLineRegex.ReplaceAllString(str, "")
		if trimmed == str {
			break
		}
		str = trimmed
	}
	str = summaryBlockRegex.ReplaceAllString(str, "")
	if loc := summaryLeakMarkerRegex.FindStringIndex(str); loc != nil {
		dropped := strings.TrimSpace(str[loc[0]:])
		head := dropped
		if len(head) > 120 {
			head = head[:120]
		}
		str = strings.TrimRight(str[:loc[0]], " \t\n\r#")
		slog.Warn("StripModelResponse summary leak stripped",
			slog.Int("dropped_chars", len(dropped)),
			slog.String("dropped_head", head))
	}
	lines := strings.Split(str, "\n")
	var fence internalUtils.FenceState
	for i, line := range lines {
		lines[i], _ = fence.Normalize(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
