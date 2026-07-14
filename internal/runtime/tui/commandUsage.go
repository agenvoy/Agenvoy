package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
)

var usagePeriods = []struct {
	label string
	days  int
}{
	{label: "24h", days: 1},
	{label: "7d", days: 7},
	{label: "28d", days: 28},
}

func (t TUI) commandUsage() (TUI, tea.Cmd, bool) {
	sessionID := strings.TrimSpace(t.currentSessionID)
	if sessionID == "" {
		return t, tea.Println(hintStyle.Render("⎯ no active session") + "\n"), true
	}

	path := filesystem.UsageLogPath(sessionID)
	now := time.Now()
	var output strings.Builder
	output.WriteString(systemStyle.Render("Usage by model"))
	output.WriteByte('\n')

	for _, period := range usagePeriods {
		summary, err := usagelog.Usage(path, period.days, now)
		if err != nil {
			if os.IsNotExist(err) {
				summary = map[string]usagelog.ModelUsage{}
			} else {
				return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] usage: %v", err)) + "\n"), true
			}
		}
		output.WriteByte('\n')
		output.WriteString(warnStyle.Render(period.label))
		output.WriteByte('\n')
		output.WriteString(renderUsageSummary(summary))
	}

	return t, tea.Println(output.String() + "\n"), true
}

func renderUsageSummary(summary map[string]usagelog.ModelUsage) string {
	if len(summary) == 0 {
		return hintStyle.Render("  no usage") + "\n"
	}

	models := make([]string, 0, len(summary))
	for model := range summary {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		left, right := summary[models[i]].Input, summary[models[j]].Input
		if left == right {
			return models[i] < models[j]
		}
		return left > right
	})

	var output strings.Builder
	output.WriteString(hintStyle.Render(fmt.Sprintf("  %-28s %12s %12s %12s", "model", "in", "out", "hit")))
	output.WriteByte('\n')
	for _, model := range models {
		u := summary[model]
		output.WriteString(fmt.Sprintf("  %-28s %12s %12s %12s\n", model, formatUsageCount(u.Input), formatUsageCount(u.Output), formatUsageCount(u.Hit)))
	}
	return output.String()
}

func formatUsageCount(value uint64) string {
	units := []struct {
		threshold uint64
		suffix    string
	}{
		{threshold: 1_000_000_000, suffix: "B"},
		{threshold: 1_000_000, suffix: "M"},
		{threshold: 1_000, suffix: "K"},
	}
	for _, unit := range units {
		if value >= unit.threshold {
			return fmt.Sprintf("%.2f%s", float64(value)/float64(unit.threshold), unit.suffix)
		}
	}
	return fmt.Sprintf("%d", value)
}
