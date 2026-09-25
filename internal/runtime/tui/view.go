package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func (t TUI) View() string {
	if t.quitting {
		return ""
	}
	if t.popup != nil {
		return t.viewPopup()
	}
	return t.viewIdle()
}

func (t TUI) viewIdle() string {
	width := t.width
	if width < 20 {
		width = 80
	}

	var fastMode string
	if fast.IsEnabled() {
		fastMode = systemStyle.Render(" fast")
	}

	var confirmMode string
	if t.allowAll {
		confirmMode = errorStyle.Render(" auto") + hintStyle.Render(" "+t.shortCwd())
	} else {
		confirmMode = okayStyle.Render(" safe") + hintStyle.Render(" "+t.shortCwd())
	}

	prefix := "\n"
	var top string
	if t.running {
		top = t.viewThinking() + "\n"
	}

	if t.selector != nil {
		top += renderCmdSelector(t.selector) + "\n"
	}

	box := textAreaStyle.Width(width - 2).Render(t.textarea.View())
	boxWidth := lipgloss.Width(box)
	bottom := hintStyle.Render(strings.Repeat("─", boxWidth))
	if model := t.modelTag(); model != "" {
		tag := " " + model + " "
		if fill := boxWidth - lipgloss.Width(tag) - 1; fill >= 1 {
			bottom = hintStyle.Render(strings.Repeat("─", fill)) + tag + hintStyle.Render("─")
		}
	}

	return prefix + top + box + "\n" + bottom + "\n" + fastMode + confirmMode
}

func (t TUI) viewThinking() string {
	var sb strings.Builder
	for _, line := range t.toolBuf {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	for _, name := range t.subOrder {
		block, ok := t.subBuf[name]
		if !ok {
			continue
		}
		sb.WriteString(block.render(name, t.width))
		sb.WriteByte('\n')
	}

	for _, line := range t.toolLog {
		sb.WriteString(hintStyle.Render(go_pkg_utils.TruncateString("  "+line, max(t.width-4, 32))))
		sb.WriteByte('\n')
	}

	verb := activityVerb(t.activity)
	elapsed := formatTime(int(time.Since(t.runStartedAt).Seconds()))

	detail := []string{elapsed}
	if in := agentTypes.FormatInput(agentTypes.InputTotals(&provider.Usage{
		Input:       t.lastIn,
		CacheRead:   t.lastCacheRead,
		CacheCreate: t.lastCacheCreate,
	})); in != "" {
		detail = append(detail, fmt.Sprintf("↑ %s ↓ %s", in, go_pkg_utils.CompactNumber(t.lastOut)))
	}
	detail = append(detail, "esc to interrupt")

	sb.WriteString(systemStyle.Render(t.spinner.View()))
	sb.WriteString(" ")
	sb.WriteString(systemStyle.Render(verb + "..."))
	sb.WriteString(" ")
	sb.WriteString(hintStyle.Render("(" + strings.Join(detail, "  ") + ")"))

	if block := renderTodoList(t.todos); block != "" {
		sb.WriteString("\n\n")
		sb.WriteString(block)
	}
	if block := renderWaitBlock(t.pendingSteer); block != "" {
		sb.WriteString("\n\n")
		sb.WriteString(block)
	}
	return sb.String()
}

func (t TUI) shortCwd() string {
	cwd := t.cwd
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		switch {
		case cwd == home:
			return "~"
		case strings.HasPrefix(cwd, home+"/"):
			return "~" + cwd[len(home):]
		}
	}
	return cwd
}

func splitOptStyle(s string) (head, tail string) {
	trimmed := strings.TrimLeft(s, " ")
	lead := len(s) - len(trimmed)
	if idx := strings.Index(trimmed, "  "); idx >= 0 {
		return s[:lead+idx], s[lead+idx:]
	}
	return s, ""
}

func renderPopupTabs(p *Popup) string {
	cells := make([]string, len(p.tabs))
	for i, tab := range p.tabs {
		label := strings.TrimSuffix(tab, "-")
		if i == p.tabIdx {
			cells[i] = systemStyle.Render("● " + label)
			continue
		}
		cells[i] = hintStyle.Render("○ " + label)
	}
	return strings.Join(cells, "   ")
}

func (t TUI) viewPopup() string {
	width := t.width
	if width < 20 {
		width = 80
	}
	p := t.popup
	if p == nil {
		return ""
	}

	head := whiteStyle.Render("⏺ " + p.title)
	if len(p.restricted) > 0 {
		head = errorStyle.Render("⚠ " + p.title)
	}
	var body []string
	if p.title != "" {
		body = append(body, head)
	}
	if p.subtitle != "" {
		body = append(body, textStyle.Render(p.subtitle))
	}
	body = append(body, p.styledLines...)
	diffWidth := max(width-6, 20)
	for _, dl := range p.diffLines {
		switch {
		case dl == "":
			body = append(body, "")
		case strings.HasPrefix(dl, "- "):
			body = append(body, diffOldStyle.Render(diffCell(dl, diffWidth)))
		default:
			body = append(body, diffNewStyle.Render(diffCell(dl, diffWidth)))
		}
	}
	trimTail := func() {
		for len(body) > 1 && strings.TrimSpace(body[len(body)-1]) == "" {
			body = body[:len(body)-1]
		}
	}
	var tail []string
	appendFooter := func(hint string) {
		if p.back != nil && p.kind != popupOAuth {
			hint = strings.NewReplacer("Esc:cancel", "Esc:back", "Esc:close", "Esc:back").Replace(hint)
		}
		trimTail()
		tail = []string{renderFooter(hint)}
	}

	trimTail()
	body = append(body, "")

	listRows := func(visible int) int {
		if t.height <= 0 {
			return visible
		}
		used := lipgloss.Height(lipgloss.NewStyle().Width(width-2).Render(strings.Join(body, "\n"))) + 2
		if len(p.tabs) > 1 {
			used += lipgloss.Height(popupStyle.Width(width).Render(renderPopupTabs(p))) + 1
		}
		if len(p.questions) > 1 {
			used++
		}
		return max(t.height-used, 1)
	}

	switch p.kind {
	case popupConfirm, popupSingleSelect:
		if p.searchable {
			p.input.SetWidth(max(width-10, 20))
			body = append(body, searchStyle.Width(max(width-8, 22)).Render(p.input.View()), "")
			if len(p.options) == 0 {
				body = append(body, hintStyle.Render("  no matching settings"))
			}
		}
		total := len(p.options)
		visible := p.maxVisible
		if visible <= 0 && p.kind == popupSingleSelect {
			visible = cmdSelectorMaxVisible
		}
		visible = listRows(visible)
		start, end := 0, total
		if visible > 0 && total > visible {
			if p.readOnly {
				start = min(p.cursor, total-visible)
				end = start + visible
			} else {
				start, end = windowRange(p.cursor, total, visible)
			}
		}
		maxLine := max(width-10, 32)
		for i := start; i < end; i++ {
			opt := truncate.StringWithTail(p.options[i], uint(maxLine), "...")
			marker := "  "
			var line string
			if !p.readOnly && i == p.cursor {
				marker = systemStyle.Render("⏵ ")
				head, tail := splitOptStyle(opt)
				line = systemStyle.Render(head)
				if tail != "" {
					line += hintStyle.Render(tail)
				}
			} else if !p.readOnly {
				head, tail := splitOptStyle(opt)
				line = whiteStyle.Render(head)
				if tail != "" {
					line += hintStyle.Render(tail)
				}
			} else {
				line = hintStyle.Render(opt)
			}
			if i < len(p.optionTail) && p.optionTail[i] != "" {
				line += " " + p.optionTail[i]
			}
			body = append(body, marker+line)
		}
		action := "confirm"
		if p.enterAction != "" {
			action = p.enterAction
		}
		hint := "Enter:" + action + "  Esc:cancel"
		if p.searchable {
			hint = "Enter:" + action + "  Esc:close"
			if p.input.Value() != "" {
				hint = "Enter:" + action + "  Esc:clear"
			}
		}
		if p.readOnly {
			hint = "Esc:close"
		}
		if p.onDelete != nil {
			hint += "  d:delete"
		}
		if p.onTag != nil {
			hint += "  t:tier tag"
		}
		if p.onMove != nil {
			hint += "  w/s:fallback order"
		}
		if p.link() != "" {
			hint += "  o:console"
		}
		appendFooter(hint)

	case popupMultiSelect:
		total := len(p.options)
		visible := p.maxVisible
		if visible <= 0 {
			visible = cmdSelectorMaxVisible
		}
		visible = listRows(visible)
		start, end := windowRange(p.cursor, total, visible)
		maxLine := max(width-14, 32)
		for i := start; i < end; i++ {
			opt := truncate.StringWithTail(p.options[i], uint(maxLine), "...")
			cursor := "  "
			head, tail := splitOptStyle(opt)
			var line string
			if i == p.cursor {
				cursor = systemStyle.Render("⏵ ")
				line = systemStyle.Render(head)
			} else {
				line = whiteStyle.Render(head)
			}
			if tail != "" {
				line += hintStyle.Render(tail)
			}
			check := "[ ]"
			if p.multi[i] {
				check = systemStyle.Render("[x]")
			}
			body = append(body, fmt.Sprintf("%s%s %s", cursor, check, line))
		}
		appendFooter("Space:toggle  Enter:confirm  Esc:cancel")

	case popupText:
		p.input.SetWidth(max(width-10, 20))
		body = append(body, p.input.View())
		if p.multiline {
			appendFooter("Ctrl+s:confirm  Enter:newline  Esc:cancel")
		} else {
			appendFooter("Enter:confirm  Esc:cancel")
		}

	case popupSecret:
		p.input.SetWidth(max(width-10, 20))
		mask := strings.Repeat("•", len([]rune(p.input.Value())))
		secret := newPopupInput(mask, false)
		cursor := p.input.LineInfo().StartColumn + p.input.LineInfo().CharOffset
		secret.SetCursor(cursor)
		secret.SetWidth(max(width-10, 20))
		body = append(body, secret.View())
		appendFooter("Enter:confirm  Esc:cancel  (input hidden)")

	case popupOAuth:
		if p.oauth != nil {
			if p.oauth.url != "" {
				body = append(body, hintStyle.Render("url:  ")+textStyle.Render(p.oauth.url))
			}
			if p.oauth.userCode != "" {
				body = append(body, hintStyle.Render("code: ")+systemStyle.Render(p.oauth.userCode))
			}
		}
		if p.oauth != nil && p.oauth.mcpServer != "" {
			appendFooter("Enter:re-open browser  p:paste redirect URL  Esc:cancel")
		} else {
			appendFooter("Enter:re-open browser  Esc:cancel")
		}
	}

	if len(p.questions) > 1 {
		footer := hintStyle.Render(fmt.Sprintf("question %d/%d", p.questionIdx+1, len(p.questions)))
		tail = append(tail, footer)
	}

	divider := hintStyle.Render(strings.Repeat("─", width))
	content := popupStyle.Width(width).Render(strings.Join(body, "\n"))
	if len(p.tabs) > 1 {
		content = popupStyle.Width(width).Render(renderPopupTabs(p)) + "\n" + divider + "\n" + content
	}
	footer, footerHeight := "", 0
	if len(tail) > 0 {
		footer = divider + "\n" + popupStyle.Width(width).Render(strings.Join(tail, "\n"))
		footerHeight = lipgloss.Height(footer)
	}

	if gap := t.height - lipgloss.Height(content) - footerHeight; t.height > 0 && gap > 0 {
		content += strings.Repeat("\n", gap)
	}
	if footer == "" {
		return content
	}
	return content + "\n" + footer
}

func (t TUI) modelTag() string {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return ""
	}

	model, reasoning := configBot.GetModel(sid)
	modelPart := hintStyle.Render(model)
	if model != configBot.DefaultModel {
		modelPart = warnStyle.Render(model)
	}

	if autoReasoningActive() {
		return modelPart
	}

	var reasonPart string
	switch reasoning {
	case "none", "low":
		reasonPart = okayStyle.Render(reasoning)
	case "high", "xhigh", "max":
		reasonPart = errorStyle.Render(reasoning)
	default:
		reasonPart = hintStyle.Render(reasoning)
	}
	return modelPart + hintStyle.Render("/") + reasonPart
}

func renderFooter(hint string) string {
	var items []string
	for item := range strings.SplitSeq(hint, "  ") {
		key, label, ok := strings.Cut(item, ":")
		if !ok {
			items = append(items, hintStyle.Render(item))
			continue
		}
		items = append(items, whiteStyle.Render(key)+hintStyle.Render(":"+label))
	}
	return strings.Join(items, hintStyle.Render(" | "))
}
