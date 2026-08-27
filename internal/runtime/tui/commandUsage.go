package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

type UsageScopeSelect struct {
	scope string
}

func (t TUI) commandUsage(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "session":
			return t.commandUsageSession()
		case "total":
			return t.commandUsageTotal()
		}
	}

	t.popup = &Popup{
		kind:  popupSingleSelect,
		title: "Usage",
		options: []string{
			"session  current session only · per-model token usage",
			"total    every session combined · includes temp / subagent / bot sessions",
		},
		values: []string{"session", "total"},
		onConfirm: func(chosen string) any {
			return UsageScopeSelect{scope: chosen}
		},
	}
	return t, nil, true
}

func (t TUI) commandUsageSession() (TUI, tea.Cmd, bool) {
	sessionID := strings.TrimSpace(t.currentSessionID)
	if sessionID == "" {
		return t, tea.Println(hintStyle.Render("⎯ no active session") + "\n"), true
	}

	return t.renderUsage("Usage by model · current session",
		func(days int, now time.Time) (map[string]usagelog.ModelUsage, error) {
			return usagelog.Usage(sessionID, days, now)
		})
}

func (t TUI) commandUsageTotal() (TUI, tea.Cmd, bool) {
	return t.renderUsage("Usage by model · all sessions", usagelog.Total)
}

func (t TUI) renderUsage(title string, load func(int, time.Time) (map[string]usagelog.ModelUsage, error)) (TUI, tea.Cmd, bool) {
	now := time.Now()

	summaries := make([]map[string]usagelog.ModelUsage, len(usagePeriods))
	for i, period := range usagePeriods {
		summary, err := load(period.days, now)
		if err != nil {
			return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] usage: %v", err)) + "\n"), true
		}
		summaries[i] = summary
	}

	var output strings.Builder
	output.WriteString(systemStyle.Render(title))
	output.WriteByte('\n')
	output.WriteString(renderUsageTable(summaries))

	return t, tea.Println(output.String() + "\n"), true
}

const usageCellWidth = 20

func renderUsageTable(summaries []map[string]usagelog.ModelUsage) string {
	seen := make(map[string]bool)
	for _, summary := range summaries {
		for model := range summary {
			seen[model] = true
		}
	}
	if len(seen) == 0 {
		return hintStyle.Render("  no usage") + "\n"
	}

	models := make([]string, 0, len(seen))
	for model := range seen {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		left, right := summaries[0][models[i]].Input, summaries[0][models[j]].Input
		if left == right {
			return models[i] < models[j]
		}
		return left > right
	})

	nameWidth := len("model")
	for _, model := range models {
		if len(model) > nameWidth {
			nameWidth = len(model)
		}
	}
	nameWidth += 1

	var output strings.Builder
	output.WriteString(hintStyle.Render(fmt.Sprintf("  %-*s", nameWidth, "model")))
	for _, period := range usagePeriods {
		output.WriteString(hintStyle.Render(fmt.Sprintf("   %-*s", usageCellWidth, period.label)))
	}
	output.WriteByte('\n')

	for _, model := range models {
		output.WriteString(fmt.Sprintf("  %-*s", nameWidth, model))
		for _, summary := range summaries {
			output.WriteString("   ")
			output.WriteString(formatUsageCell(summary[model]))
		}
		output.WriteByte('\n')
	}
	return output.String()
}

func formatUsageCell(u usagelog.ModelUsage) string {
	if u.Input == 0 && u.Output == 0 {
		return strings.Repeat(" ", usageCellWidth)
	}
	hitPct := 0.0
	if total := u.Input + u.Hit; total > 0 {
		hitPct = float64(u.Hit) / float64(total) * 100
	}
	rounded := int(hitPct + 0.5)
	var pct string
	switch {
	case rounded <= 0:
		pct = "--%"
	case rounded >= 100:
		pct = "00%"
	default:
		pct = fmt.Sprintf("%2d%%", rounded)
	}
	return fmt.Sprintf("%s(%s)/%s", color(u.Input), pct, color(u.Output))
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

func color(value uint64) string {
	plain := fmt.Sprintf("%7s", formatUsageCount(value))
	switch {
	case value >= 1_000_000_000:
		return errorStyle.Render(plain)
	case value >= 1_000_000:
		return systemStyle.Render(plain)
	case value >= 1_000:
		return okayStyle.Render(plain)
	default:
		return plain
	}
}
