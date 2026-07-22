package tui

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

var (
	mdHeadingRe    = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)`)
	mdBoldRe       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdItalicRe     = regexp.MustCompile(`\*([^\s*](?:[^*]*[^\s*])?)\*`)
	mdBlockquoteRe = regexp.MustCompile(`(?m)^>\s?(.*)`)
	htmlTagRe      = regexp.MustCompile(`<[^>]*>`)
)

const defaultWrapWidth = 80

func wordWrap(s string, width int) string {
	f := &wordwrap.WordWrap{Limit: width, Newline: []rune{'\n'}, KeepNewlines: true}
	_, _ = f.Write([]byte(s))
	_ = f.Close()
	return f.String()
}

func wrapText(s string, width int) string {
	if width <= 0 {
		width = defaultWrapWidth
	}
	width = max(width, 10)
	return wrapLines(wordWrap(s, width), width)
}

func wrapLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = wrap.String(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

const tableBoxChars = "┌┬┐├┼┤└┴┘│─"

func wrapProse(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.ContainsAny(line, tableBoxChars) {
			continue
		}
		lines[i] = wrapText(line, width)
	}
	return strings.Join(lines, "\n")
}

func toPureText(s string) string {
	s = mdBoldRe.ReplaceAllString(s, "$1")
	s = htmlTagRe.ReplaceAllString(s, "")
	return s
}

func renderMarkdown(s string, width int) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = renderTables(s, width)
	s = mdBlockquoteRe.ReplaceAllStringFunc(s, func(match string) string {
		m := mdBlockquoteRe.FindStringSubmatch(match)
		content := mdBoldRe.ReplaceAllString(m[1], "$1")
		content = mdItalicRe.ReplaceAllString(content, "$1")
		return systemStyle.Render("▎ " + content)
	})
	s = mdBoldRe.ReplaceAllStringFunc(s, func(match string) string {
		return userStyle.Bold(true).Render(match[2 : len(match)-2])
	})
	s = mdItalicRe.ReplaceAllString(s, "$1")
	s = mdHeadingRe.ReplaceAllStringFunc(s, func(match string) string {
		m := mdHeadingRe.FindStringSubmatch(match)
		return okayStyle.Bold(true).Render(m[2])
	})
	return s
}

func isTableSep(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return false
	}
	dashes := 0
	for _, c := range trimmed {
		switch c {
		case '-':
			dashes++
		case '|', ':', ' ':
		default:
			return false
		}
	}
	return dashes >= 3
}

func splitTableRow(line string) []string {
	runes := []rune(line)
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(runes); i++ {
		switch {
		case runes[i] == '\\' && i+1 < len(runes) && runes[i+1] == '|':
			cur.WriteRune('|')
			i++
		case runes[i] == '|':
			cells = append(cells, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(runes[i])
		}
	}
	cells = append(cells, cur.String())
	return cells
}

func parseTableCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := splitTableRow(line)
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func cleanTableCell(s string) string {
	s = mdBoldRe.ReplaceAllString(s, "$1")
	s = mdItalicRe.ReplaceAllString(s, "$1")
	return s
}

func renderTables(s string, width int) string {
	lines := strings.Split(s, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		if i+1 < len(lines) && strings.Contains(lines[i], "|") && isTableSep(lines[i+1]) {
			end := i + 2
			for end < len(lines) && strings.Contains(lines[end], "|") && !isTableSep(lines[end]) {
				end++
			}
			header := parseTableCells(lines[i])
			var rows [][]string
			for j := i + 2; j < end; j++ {
				rows = append(rows, parseTableCells(lines[j]))
			}
			out = append(out, buildTable(header, rows, width))
			i = end
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

const tableColMinWidth = 3

func buildTable(header []string, rows [][]string, termWidth int) string {
	numCols := len(header)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	for len(header) < numCols {
		header = append(header, "")
	}
	for i := range rows {
		for len(rows[i]) < numCols {
			rows[i] = append(rows[i], "")
		}
	}

	for i := range header {
		header[i] = cleanTableCell(header[i])
	}
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] = cleanTableCell(rows[i][j])
		}
	}

	natural := make([]int, numCols)
	for i, h := range header {
		if w := lipgloss.Width(h); w > natural[i] {
			natural[i] = w
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := lipgloss.Width(cell); w > natural[i] {
				natural[i] = w
			}
		}
	}

	if termWidth <= 0 {
		termWidth = defaultWrapWidth
	}
	overhead := numCols*3 + 1
	available := termWidth - overhead

	totalNatural := 0
	for _, w := range natural {
		totalNatural += w
	}

	widths := make([]int, numCols)
	if available <= 0 || totalNatural <= available {
		copy(widths, natural)
	} else {
		for i, w := range natural {
			widths[i] = max(w*available/totalNatural, tableColMinWidth)
		}
	}

	b := func(s string) string { return hintStyle.Render(s) }

	var sb strings.Builder

	hLine := func(left, mid, right string) {
		sb.WriteString(b(left))
		for i, w := range widths {
			sb.WriteString(b(strings.Repeat("─", w+2)))
			if i < numCols-1 {
				sb.WriteString(b(mid))
			}
		}
		sb.WriteString(b(right))
	}

	writeRow := func(cells []string, bold bool) {
		cellLines := make([][]string, numCols)
		maxLines := 1
		for i, cell := range cells {
			lines := strings.Split(wrapLines(wordWrap(cell, widths[i]), widths[i]), "\n")
			cellLines[i] = lines
			maxLines = max(maxLines, len(lines))
		}
		for ln := 0; ln < maxLines; ln++ {
			if ln > 0 {
				sb.WriteByte('\n')
			}
			for i, w := range widths {
				var content string
				if ln < len(cellLines[i]) {
					content = cellLines[i][ln]
				}
				sb.WriteString(b("│"))
				pad := max(w-lipgloss.Width(content), 0)
				if bold {
					left := pad / 2
					right := pad - left
					sb.WriteString(strings.Repeat(" ", left+1))
					sb.WriteString(whiteStyle.Bold(true).Render(content))
					sb.WriteString(strings.Repeat(" ", right+1))
				} else {
					sb.WriteByte(' ')
					sb.WriteString(content)
					sb.WriteString(strings.Repeat(" ", pad+1))
				}
			}
			sb.WriteString(b("│"))
		}
	}

	hLine("┌", "┬", "┐")
	sb.WriteByte('\n')
	writeRow(header, true)
	sb.WriteByte('\n')
	hLine("├", "┼", "┤")

	for i, row := range rows {
		sb.WriteByte('\n')
		writeRow(row, false)
		if i < len(rows)-1 {
			sb.WriteByte('\n')
			hLine("├", "┼", "┤")
		}
	}

	sb.WriteByte('\n')
	hLine("└", "┴", "┘")

	return sb.String()
}

var projectVersion = "dev"

var (
	headerStyle = lipgloss.NewStyle()

	textAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(colHint).
			Padding(0, 1)

	popupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colWarn).
			Padding(0, 1)
)

// * one row per line of headerBlock's body; top half reads as "A", bottom half as "V"
var asciiMarkLines = []string{
	"      .:::::.",
	"    .::     ::.",
	"  .::    :::::::.",
	"",
	"  ::.         .::",
	"    ::.     .::",
	"      :::::::",
}

func headerBlock(daemon, http, discord, telegram, line string) string {
	logo := whiteStyle.Bold(true).Render("Agenvoy ") + hintStyle.Render(projectVersion)

	const markCol = 20
	const gap = "   "
	padTo := func(s string, width int) string {
		w := lipgloss.Width(s)
		if w >= width {
			return s
		}
		return s + strings.Repeat(" ", width-w)
	}

	textLines := []string{
		logo,
		hintStyle.Render("Make AI actually work for you"),
		hintStyle.Render("Your productivity infrastructure"),
		"",
		daemon + gap + discord,
		http + gap + telegram,
		line,
	}

	rows := make([]string, len(asciiMarkLines))
	for i, mark := range asciiMarkLines {
		rows[i] = whiteStyle.Render(padTo(mark, markCol)) + textLines[i]
	}
	rows = append(rows, "")
	return headerStyle.Render(strings.Join(rows, "\n"))
}

func messageBlock(str string) string {
	var sb strings.Builder
	for i, line := range strings.Split(str, "\n") {
		if i > 0 {
			sb.WriteString("\n  ")
		} else {
			sb.WriteString(hintStyle.Render("❯ "))
		}
		sb.WriteString(userStyle.Render(line))
	}
	return sb.String()
}

func shellEchoBlock(str string) string {
	var sb strings.Builder
	for i, line := range strings.Split(str, "\n") {
		if i > 0 {
			sb.WriteString("\n  ")
		} else {
			sb.WriteString(warnStyle.Render("$ "))
		}
		sb.WriteString(userStyle.Render(line))
	}
	return sb.String()
}

func thinkingBlock(str string) string {
	var sb strings.Builder
	for i, line := range strings.Split(str, "\n") {
		if i > 0 {
			sb.WriteString("\n  ")
		} else {
			sb.WriteString(whiteStyle.Render("✻ "))
		}
		sb.WriteString(whiteStyle.Render(line))
	}
	return sb.String()
}

func messageRow(text, subagent string) string {
	prefix := systemStyle.Render("⏺ ")
	if strings.TrimSpace(subagent) != "" {
		prefix = warnStyle.Render("⏺ [" + subagent + "] ")
	}
	indent := "  "

	var sb strings.Builder
	first := true
	for line := range strings.SplitSeq(text, "\n") {
		if first {
			sb.WriteString(prefix)
			sb.WriteString(line)
			first = false
			continue
		}
		sb.WriteByte('\n')
		sb.WriteString(indent)
		sb.WriteString(line)
	}
	return sb.String()
}

// * context for live usage, context = nil for replay
func renderAgentEvent(ctx context.Context, ev agentTypes.Event, sessionLabel, cwd string, width int, finishedAt string) (string, bool) {
	src := strings.TrimSpace(ev.Source)
	srcPrefix := ""
	if src != "" {
		srcPrefix = "[" + src + "] "
	}

	switch ev.Type {
	case agentTypes.EventAgentSelect:
		if ev.Source == "" {
			return "", false
		}
		return hintStyle.Render("  ⎿ " + srcPrefix + "selecting agent…"), true

	case agentTypes.EventAgentResult:
		return "", false

	case agentTypes.EventToolCall:
		if ev.ToolName == "ask_user" || ev.ToolName == "store_secret" ||
			ev.ToolName == "write_todo" {
			return "", false
		}
		bullet := "⏵"
		if ev.Source != "" {
			bullet = "  ⎿"
		}
		return buildToolLine(bullet, ev.Source, ev.ToolName, ev.ToolArgs, cwd, width), true

	case agentTypes.EventToolSkipped:
		line := "  ⎿ " + srcPrefix + "skipped: " + ev.ToolName
		if arg := utils.FormatToolArgs(ev.ToolName, ev.ToolArgs, cwd); arg != "" {
			line += "(" + arg + ")"
		}
		return hintStyle.Render(line), true

	case agentTypes.EventText:
		if ev.Source != "" {
			str := toPureText(ev.Text)
			if str == "" {
				return "", false
			}
			return hintStyle.Render("  ⎿ " + srcPrefix + oneLine(str)), true
		}
		str := renderMarkdown(ev.Text, width-2)
		if str == "" {
			return "", false
		}
		return messageRow(wrapProse(str, width-2), sessionLabel), true

	case agentTypes.EventReasoning:
		if src != "" {
			str := oneLine(toPureText(ev.Text))
			if str == "" {
				return "", false
			}
			return hintStyle.Render("  ⎿ " + srcPrefix + "✻ " + str), true
		}
		var kept []string
		for line := range strings.SplitSeq(toPureText(ev.Text), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			kept = append(kept, strings.TrimRight(line, " \t"))
		}
		if len(kept) == 0 {
			return "", false
		}
		return thinkingBlock(wrapText(strings.Join(kept, "\n"), width-2)), true

	case agentTypes.EventUserInjected:
		text := strings.TrimSpace(ev.Text)
		if text == "" {
			return "", false
		}
		return hintStyle.Render("❯ ") + okayStyle.Render(text), true

	case agentTypes.EventExecError:
		return errorStyle.Render("  ⎿ " + srcPrefix + "error: " + ev.ToolName + " — " + ev.Text), true

	case agentTypes.EventError:
		if ev.Err == nil {
			return "", false
		}
		return errorStyle.Render("  ⎿ " + srcPrefix + fmt.Sprintf("error: %v", ev.Err)), true

	case agentTypes.EventSummaryGenerate:
		return hintStyle.Render("⏵ " + srcPrefix + "summarizing…"), true

	case agentTypes.EventCompact:
		label := "compacting tool history…"
		if ev.Text == "history" {
			label = "compacting conversation history…"
		}
		return hintStyle.Render("⏵ " + srcPrefix + label), true

	case agentTypes.EventDone:
		var footer string
		if ctx != nil {
			footer = utils.FormatEventFooterContext(ctx, ev.Duration, ev.Model, ev.Usage)
		} else {
			footer = utils.FormatEventFooter(ev.Duration, ev.Model, ev.Usage)
		}
		if sessionLabel != "" {
			if footer != "" {
				footer = footer + " · [" + sessionLabel + "]"
			} else {
				footer = "[" + sessionLabel + "]"
			}
		}
		if finishedAt != "" {
			if footer != "" {
				footer = footer + " · " + finishedAt
			} else {
				footer = finishedAt
			}
		}
		if footer == "" {
			return "", false
		}
		return hintStyle.Render("  ⎿ "+footer) + "\n", true

	case agentTypes.EventCanceled:
		footer := "canceled"
		if finishedAt != "" {
			footer += " · " + finishedAt
		}
		return warnStyle.Render("  ⎿ "+footer) + "\n", true
	}

	return "", false
}

var (
	diffOldStyle = lipgloss.NewStyle().Background(lipgloss.Color("#400000")).Foreground(lipgloss.Color("#FFFFFF"))
	diffNewStyle = lipgloss.NewStyle().Background(lipgloss.Color("#002a00")).Foreground(lipgloss.Color("#FFFFFF"))
)

func rowLabel(row, offset int) string {
	if row <= 0 {
		return ""
	}
	return strconv.Itoa(row+offset) + " "
}

func padToWidth(s string, width int) string {
	if w := lipgloss.Width(s); width > w {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func buildToolLine(bullet, source, name, args, cwd string, width int) string {
	if width <= 0 {
		width = defaultWrapWidth
	}
	src := strings.TrimSpace(source)
	srcPrefix := ""
	if src != "" {
		srcPrefix = "[" + src + "] "
	}
	line := bullet + " " + srcPrefix + utils.ToolName(name)
	if arg := utils.FormatToolArgs(name, args, cwd); arg != "" {
		line += "(" + arg + ")"
	}
	style := hintStyle
	if name == "invoke_subagent" {
		style = lipgloss.NewStyle().Foreground(colOk)
	}
	header := style.Render(line)

	switch name {
	case "patch_file", "patch_tool", "patch_skill":
		hunks := utils.FormatPatchDiff(args)
		if len(hunks) == 0 {
			return header
		}
		var sb strings.Builder
		sb.WriteString(header)
		remaining := 32
		for i, h := range hunks {
			if remaining <= 0 {
				break
			}
			if i > 0 {
				sb.WriteByte('\n')
			}
			for j, l := range h.OldLines[:min(len(h.OldLines), 16, remaining)] {
				sb.WriteByte('\n')
				sb.WriteString(diffOldStyle.Render(padToWidth("  - "+rowLabel(h.Row, j)+l, width)))
				remaining--
			}
			for j, l := range h.NewLines[:min(len(h.NewLines), remaining)] {
				sb.WriteByte('\n')
				sb.WriteString(diffNewStyle.Render(padToWidth("  + "+rowLabel(h.Row, j)+l, width)))
				remaining--
			}
		}
		return sb.String()

	case "write_file":
		lines := utils.FormatWriteDiff(args)
		if len(lines) == 0 {
			return header
		}
		var sb strings.Builder
		sb.WriteString(header)
		for _, l := range lines[:min(len(lines), 16)] {
			sb.WriteByte('\n')
			sb.WriteString(diffNewStyle.Render(padToWidth("  + "+l, width)))
		}
		return sb.String()
	}

	return header
}

func oneLine(s string) string {
	r := strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ")
	return r.Replace(s)
}
