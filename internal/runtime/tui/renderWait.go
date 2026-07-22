package tui

import (
	"fmt"
	"strings"
)

func renderWaitBlock(queued []string) string {
	if len(queued) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(okayStyle.Render("⏺ Steer"))
	sb.WriteString(hintStyle.Render(fmt.Sprintf(" (%d)", len(queued))))
	for _, q := range queued {
		sb.WriteByte('\n')
		sb.WriteString(hintStyle.Render("  ● " + q))
	}
	return sb.String()
}
