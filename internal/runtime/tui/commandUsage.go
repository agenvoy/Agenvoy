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

	labels := make([]string, len(usagePeriods))
	for i, period := range usagePeriods {
		labels[i] = period.label
	}

	nameWidth := usageNameWidth(summaries)
	popup := &Popup{
		kind:       popupSingleSelect,
		title:      title,
		subtitle:   hintStyle.Render("  input(cache hit%)/output"),
		maxVisible: usageMaxVisible,
		tabs:       labels,
	}
	popup.onTab = func(p *Popup) {
		fillUsageOptions(p, summaries, nameWidth)
	}
	popup.onTab(popup)

	t.popup = popup
	return t, nil, true
}

const usageMaxVisible = 16

func usageNameWidth(summaries []map[string]usagelog.ModelUsage) int {
	width := len("model")
	for _, summary := range summaries {
		for model, one := range summary {
			if one.Input == 0 && one.Output == 0 {
				continue
			}
			if len(model) > width {
				width = len(model)
			}
		}
	}
	return width
}

func fillUsageOptions(p *Popup, summaries []map[string]usagelog.ModelUsage, nameWidth int) {
	idx := p.tabIdx
	if idx < 0 || idx >= len(summaries) {
		idx = 0
	}
	summary := summaries[idx]

	models := make([]string, 0, len(summary))
	for model, one := range summary {
		if one.Input == 0 && one.Output == 0 {
			continue
		}
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		left, right := summary[models[i]].Input, summary[models[j]].Input
		if left == right {
			return models[i] < models[j]
		}
		return left > right
	})

	options := make([]string, 0, len(models))
	tails := make([]string, 0, len(models))
	for _, model := range models {
		options = append(options, fmt.Sprintf("%-*s", nameWidth, model))
		tails = append(tails, formatUsageCell(summary[model]))
	}
	if len(options) == 0 {
		options = append(options, "no usage")
		tails = append(tails, "")
	}

	p.options = options
	p.optionTail = tails
	p.values = nil
	p.cursor = 0
}

func formatUsageCell(u usagelog.ModelUsage) string {
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
