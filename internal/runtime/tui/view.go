package tui

import (
	"fmt"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	"github.com/pardnchiu/agenvoy/internal/utils"
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
		fastMode = systemStyle.Render(" [fast]")
	}

	var confirmMode string
	if t.allowAll {
		confirmMode = errorStyle.Render(" [auto]") + hintStyle.Render(" "+t.shortCwd())
	} else {
		confirmMode = okayStyle.Render(" [safe]") + hintStyle.Render(" "+t.shortCwd())
	}
	right := t.sessionTag()

	prefix := "\n"
	var top string
	if t.running {
		top = t.viewThinking() + "\n\n"
	}

	if t.selector != nil {
		top += renderCmdSelector(t.selector) + "\n"
	}

	box := textAreaStyle.Width(width - 2).Render(t.textarea.View())

	left := fastMode + confirmMode
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	pad = max(pad, 1)
	return prefix + top + box + "\n" + left + strings.Repeat(" ", pad) + right
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
	if t.currentModel != "" {
		detail = append(detail, t.currentModel)
	}
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
			cells[i] = systemStyle.Render("[" + label + "]")
			continue
		}
		cells[i] = hintStyle.Render(" " + label + " ")
	}
	return strings.Join(cells, " ")
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
	body := []string{head}
	if p.subtitle != "" {
		body = append(body, textStyle.Render(p.subtitle))
	}
	body = append(body, p.styledLines...)
	if len(p.tabs) > 1 {
		body = append(body, "", "  "+renderPopupTabs(p))
	}
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
	appendFooter := func(hint string) {
		if p.back != nil && p.kind != popupOAuth {
			hint = strings.NewReplacer("esc cancel", "esc back", "esc close", "esc back").Replace(hint)
		}
		trimTail()
		body = append(body, "", hintStyle.Render(hint))
	}

	trimTail()
	body = append(body, "")

	switch p.kind {
	case popupConfirm, popupSingleSelect:
		total := len(p.options)
		visible := p.maxVisible
		if visible <= 0 && p.kind == popupSingleSelect {
			visible = cmdSelectorMaxVisible
		}
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
				marker = systemStyle.Render("> ")
				head, tail := splitOptStyle(opt)
				line = systemStyle.Render(head)
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
		hint := "↑/↓ select  enter " + action + "  esc cancel"
		if len(p.tabs) > 1 {
			hint = "↑/↓ select  ←/→ filter  enter " + action + "  esc cancel"
		}
		if p.readOnly {
			hint = "↑/↓ scroll  esc close"
			if len(p.tabs) > 1 {
				hint = "↑/↓ scroll  ←/→ filter  esc close"
			}
		}
		if p.onDelete != nil {
			hint += "  d delete"
		}
		if p.onTag != nil {
			hint += "  t tag"
		}
		appendFooter(hint)

	case popupMultiSelect:
		total := len(p.options)
		visible := p.maxVisible
		if visible <= 0 {
			visible = cmdSelectorMaxVisible
		}
		start, end := windowRange(p.cursor, total, visible)
		maxLine := max(width-14, 32)
		for i := start; i < end; i++ {
			opt := truncate.StringWithTail(p.options[i], uint(maxLine), "...")
			cursor := "  "
			head, tail := splitOptStyle(opt)
			var line string
			if i == p.cursor {
				cursor = systemStyle.Render("> ")
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
		if len(p.tabs) > 1 {
			appendFooter("↑/↓ move  ←/→ filter  space toggle  enter confirm  esc cancel")
		} else {
			appendFooter("↑/↓ move  space toggle  enter confirm  esc cancel")
		}

	case popupText:
		p.input.SetWidth(max(width-10, 20))
		body = append(body, p.input.View())
		if p.multiline {
			appendFooter("ctrl+s confirm  enter newline  esc cancel")
		} else {
			appendFooter("enter confirm  esc cancel")
		}

	case popupSecret:
		p.input.SetWidth(max(width-10, 20))
		mask := strings.Repeat("•", len([]rune(p.input.Value())))
		secret := newPopupInput(mask, false)
		cursor := p.input.LineInfo().StartColumn + p.input.LineInfo().CharOffset
		secret.SetCursor(cursor)
		secret.SetWidth(max(width-10, 20))
		body = append(body, secret.View())
		appendFooter("enter confirm  esc cancel  (input hidden)")

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
			appendFooter("enter re-open browser  p paste redirect URL  esc cancel")
		} else {
			appendFooter("enter re-open browser  esc cancel")
		}
	}

	if len(p.questions) > 1 {
		footer := hintStyle.Render(fmt.Sprintf("question %d/%d", p.questionIdx+1, len(p.questions)))
		body = append(body, footer)
	}

	return popupStyle.Width(width - 4).Render(strings.Join(body, "\n"))
}

func (t TUI) sessionTag() string {
	var parts []string
	if name := t.sessionName(); name != "" {
		parts = append(parts, hintStyle.Render(name))
	}
	return strings.Join(parts, hintStyle.Render("  ")) + hintStyle.Render("  ")
}

func (t TUI) sessionName() string {
	sid := strings.TrimSpace(t.currentSessionID)
	name := strings.TrimSpace(t.currentSessionName)
	if sid == "" {
		return hintStyle.Render("(no session)")
	}

	short := utils.ShortenSessionID(sid)
	base := short
	if name != "" && name != sid {
		base = fmt.Sprintf("%s (%s)", name, short)
	}

	model, reasoning := configBot.GetModel(sid)
	modelPart := hintStyle.Render(model)
	if model != configBot.DefaultModel {
		modelPart = warnStyle.Render(model)
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
	return base + hintStyle.Render(" (") + modelPart + hintStyle.Render("/") + reasonPart + hintStyle.Render(")")
}
