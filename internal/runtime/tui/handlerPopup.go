package tui

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	goruntime "runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

type popupType int

const (
	popupConfirm popupType = iota
	popupText
	popupSecret
	popupSingleSelect
	popupMultiSelect
	popupOAuth
)

type Popup struct {
	pendingId string

	kind        popupType
	title       string
	subtitle    string
	styledLines []string
	diffLines   []string

	options    []string
	optionTail []string
	values     []string
	cursor     int
	multi      map[int]bool
	onToggle   func(p *Popup, index int)
	maxVisible int

	tabs   []string
	tabIdx int
	onTab  func(p *Popup)

	input          textarea.Model
	multiline      bool
	skipWithReason bool
	restricted     []string

	questions   []runtime.Question
	questionIdx int
	answers     []any

	onConfirm func(chosen string) any

	oauth *oauthState
}

type oauthState struct {
	provider  string
	mcpServer string
	url       string
	userCode  string
	cancel    context.CancelFunc
}

func (t TUI) closePopup() TUI {
	t.popup = nil
	for len(t.popupQueue) > 0 {
		next := t.popupQueue[0]
		t.popupQueue = t.popupQueue[1:]
		if ps := newPopup(next.id, next.request); ps != nil {
			t.popup = ps
			return t
		}
		runtime.Resolve(next.id, runtime.Reply{
			Error: fmt.Errorf("invalid pending request"),
		})
	}
	return t
}

func (t TUI) updatePopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch t.popup.kind {
	case popupConfirm:
		return t.updateConfirmPopup(msg)

	case popupSingleSelect:
		return t.updateSingleSelectPopup(msg)

	case popupMultiSelect:
		return t.updateMultiSelectPopup(msg)

	case popupText, popupSecret:
		return t.updateTextInputPopup(msg)

	case popupOAuth:
		return t.updateOAuthPopup(msg)
	}
	return t, nil
}

func (t TUI) updateOAuthPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := t.popup
	if p.oauth == nil {
		return t, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		if p.oauth.cancel != nil {
			p.oauth.cancel()
		}
	case tea.KeyEnter:
		if p.oauth.url != "" {
			openBrowser(p.oauth.url)
		}
	case tea.KeyRunes:
		if p.oauth.mcpServer != "" && strings.EqualFold(string(msg.Runes), "p") {
			return t.openMcpOAuthPaste(p.oauth)
		}
	}
	return t, nil
}

func openBrowser(link string) {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", link)
	case "linux":
		cmd = linuxOpenCommand(link)
	default:
		return
	}
	if cmd == nil {
		slog.Warn("openOAuthBrowser: no opener available",
			slog.String("url", link))
		return
	}
	if err := cmd.Start(); err != nil {
		slog.Warn("openOAuthBrowser cmd.Start",
			slog.String("url", link),
			slog.String("error", err.Error()))
	}
}

func linuxOpenCommand(link string) *exec.Cmd {
	if bin := utils.WSLChromePath(); bin != "" {
		return exec.Command(bin, link)
	}
	if bin, err := exec.LookPath("wslview"); err == nil {
		return exec.Command(bin, link)
	}
	if bin, err := exec.LookPath("xdg-open"); err == nil {
		return exec.Command(bin, link)
	}
	return nil
}

func (t TUI) updateConfirmPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := t.popup
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		p.cursor = (p.cursor - 1 + len(p.options)) % len(p.options)

	case tea.KeyDown, tea.KeyTab:
		p.cursor = (p.cursor + 1) % len(p.options)

	case tea.KeyEsc:
		runtime.Resolve(p.pendingId, runtime.Reply{
			Approve: false,
			Error:   fmt.Errorf("user stopped"),
		})
		t = t.closePopup()

	case tea.KeyEnter:
		chosen := p.options[p.cursor]
		var reply runtime.Reply
		switch {
		case chosen == "Yes":
			if len(p.restricted) > 0 {
				id := p.pendingId
				t = t.closePopup()
				if sudoCached() {
					return t, func() tea.Msg { return RestrictedAuthDone{pendingID: id, cached: true} }
				}
				return t, tea.Sequence(
					tea.Println(warnStyle.Render("⎯ restricted path: system password required")+"\n"),
					tea.ExecProcess(exec.Command("sudo", "-v"), func(err error) tea.Msg {
						return RestrictedAuthDone{pendingID: id, err: err}
					}),
				)
			}
			reply = runtime.Reply{Approve: true}
		case strings.HasPrefix(chosen, "Yes  don't ask again"):
			reply = runtime.Reply{Approve: true, Remember: true}
		case strings.HasPrefix(chosen, "Yes  allow this turn"):
			reply = runtime.Reply{Approve: true, AllowTurn: true}
		case chosen == "No":
			p.kind = popupText
			p.skipWithReason = true
			p.title = "Reason (enter to skip):"
			p.input = newPopupInput("", false)
			return t, nil
		case chosen == "Abort task":
			reply = runtime.Reply{Approve: false, Error: fmt.Errorf("user stopped")}
		}
		runtime.Resolve(p.pendingId, reply)
		t = t.closePopup()
	}
	return t, nil
}

func (t TUI) updateSingleSelectPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := t.popup
	switch msg.Type {
	case tea.KeyUp:
		p.cursor = (p.cursor - 1 + len(p.options)) % len(p.options)

	case tea.KeyDown:
		p.cursor = (p.cursor + 1) % len(p.options)

	case tea.KeyLeft:
		p.switchTab(-1)

	case tea.KeyRight:
		p.switchTab(1)

	case tea.KeyEsc:
		if p.pendingId == "" {
			t = t.closePopup()
		} else {
			runtime.Resolve(p.pendingId, runtime.Reply{
				Error: fmt.Errorf("user cancelled"),
			})
			t = t.closePopup()
		}
	case tea.KeyEnter:
		chosen := p.options[p.cursor]
		if p.values != nil && p.cursor < len(p.values) {
			chosen = p.values[p.cursor]
		}
		if p.pendingId == "" {
			cb := p.onConfirm
			t = t.closePopup()
			if cb == nil {
				return t, nil
			}
			return t, func() tea.Msg { return cb(chosen) }
		}

		resolved, reply := p.advanceOrResolve(chosen)
		if resolved {
			runtime.Resolve(p.pendingId, reply)
			t = t.closePopup()
		}
	}
	return t, nil
}

func (p *Popup) switchTab(step int) {
	if len(p.tabs) < 2 || p.onTab == nil {
		return
	}
	p.tabIdx = (p.tabIdx + step + len(p.tabs)) % len(p.tabs)
	p.onTab(p)
}

func (t TUI) updateMultiSelectPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := t.popup
	switch msg.Type {
	case tea.KeyUp:
		p.cursor = (p.cursor - 1 + len(p.options)) % len(p.options)

	case tea.KeyDown:
		p.cursor = (p.cursor + 1) % len(p.options)

	case tea.KeyLeft:
		p.switchTab(-1)

	case tea.KeyRight:
		p.switchTab(1)

	case tea.KeySpace:
		p.multi[p.cursor] = !p.multi[p.cursor]
		if p.onToggle != nil {
			p.onToggle(p, p.cursor)
		}

	case tea.KeyEsc:
		if p.pendingId == "" {
			t = t.closePopup()
		} else {
			runtime.Resolve(p.pendingId, runtime.Reply{
				Error: fmt.Errorf("user cancelled"),
			})
			t = t.closePopup()
		}

	case tea.KeyEnter:
		selected := make([]string, 0, len(p.multi))
		for i := range p.options {
			if p.multi[i] {
				v := p.options[i]
				if p.values != nil && i < len(p.values) {
					v = p.values[i]
				}
				selected = append(selected, v)
			}
		}

		if p.pendingId == "" {
			cb := p.onConfirm
			t = t.closePopup()
			if cb == nil {
				return t, nil
			}
			return t, func() tea.Msg { return cb(strings.Join(selected, "\x1F")) }
		}

		resolved, reply := p.advanceOrResolve(selected)
		if resolved {
			runtime.Resolve(p.pendingId, reply)
			t = t.closePopup()
		}
	}
	return t, nil
}

func newPopupInput(value string, multiline bool) textarea.Model {
	input := textarea.New()
	input.CharLimit = 8192
	input.ShowLineNumbers = false
	input.SetHeight(1)
	input.SetValue(value)
	input.Focus()
	input.Cursor.Style = whiteStyle
	input.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return systemStyle.Render("> ")
		}
		return "  "
	})
	if multiline {
		input.SetHeight(6)
	}
	return input
}

func (t TUI) updateTextInputPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := t.popup
	submit := func() (tea.Model, tea.Cmd) {
		value := p.input.Value()
		if p.pendingId == "" {
			cb := p.onConfirm
			t = t.closePopup()
			if cb == nil {
				return t, nil
			}
			return t, func() tea.Msg { return cb(value) }
		}
		if p.skipWithReason {
			runtime.Resolve(p.pendingId, runtime.Reply{
				Approve: false,
				Skip:    true,
				Reason:  strings.TrimSpace(value),
			})
			t = t.closePopup()
			return t, nil
		}
		resolved, reply := p.advanceOrResolve(value)
		if resolved {
			runtime.Resolve(p.pendingId, reply)
			t = t.closePopup()
		}
		return t, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		if p.pendingId == "" {
			t = t.closePopup()
			return t, nil
		}
		runtime.Resolve(p.pendingId, runtime.Reply{
			Error: fmt.Errorf("user cancelled"),
		})
		t = t.closePopup()
		return t, nil

	case tea.KeyCtrlS:
		if p.multiline {
			return submit()
		}

	case tea.KeyEnter:
		if !p.multiline {
			return submit()
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return t, cmd
}

func newPopup(id string, req runtime.Request) *Popup {
	switch req.Kind {
	case runtime.KindToolConfirm:
		display := utils.FormatToolEvent(req.ToolName, req.ToolArgs)
		if display == "" {
			display = req.ToolArgs
		}
		options := []string{
			"Yes",
			"Yes  don't ask again",
			"Yes  allow this turn",
			"No",
			"Abort task",
		}
		if len(req.Restricted) > 0 {
			options = []string{
				"Yes",
				"No",
				"Abort task",
			}
		}
		p := &Popup{
			pendingId:  id,
			kind:       popupConfirm,
			title:      fmt.Sprintf("Run %s?", utils.ToolName(req.ToolName)),
			subtitle:   display,
			options:    options,
			restricted: req.Restricted,
		}
		if len(req.Restricted) > 0 {
			p.title = fmt.Sprintf("Run %s outside the allow list?", utils.ToolName(req.ToolName))
			for _, one := range req.Restricted {
				p.styledLines = append(p.styledLines, warnStyle.Render("⚠ "+one))
			}
			p.styledLines = append(p.styledLines, "")
			if sudoCached() {
				p.styledLines = append(p.styledLines, okayStyle.Render("sudo credentials still valid — no password needed"))
			} else {
				p.styledLines = append(p.styledLines, userStyle.Render("system password required — you will be prompted after Yes"))
			}
		}
		switch req.ToolName {
		case "edit_file", "edit_tool", "edit_skill":
			hunks := utils.FormatPatchDiff(req.ToolArgs)
			if len(hunks) == 0 {
				for _, l := range utils.FormatWriteDiff(req.ToolArgs) {
					p.diffLines = append(p.diffLines, "+ "+l)
					if len(p.diffLines) >= 16 {
						break
					}
				}
			}
			remaining := 32
			for i, h := range hunks {
				if remaining <= 0 {
					break
				}
				if i > 0 {
					p.diffLines = append(p.diffLines, "")
				}
				for j, l := range h.OldLines[:min(len(h.OldLines), 16, remaining)] {
					p.diffLines = append(p.diffLines, "- "+rowLabel(h.Row, j)+l)
					remaining--
				}
				for j, l := range h.NewLines[:min(len(h.NewLines), remaining)] {
					p.diffLines = append(p.diffLines, "+ "+rowLabel(h.Row, j)+l)
					remaining--
				}
			}
		}
		return p
	case runtime.KindAskUser:
		if req.AskUser == nil || len(req.AskUser.Questions) == 0 {
			return nil
		}
		ps := &Popup{
			pendingId:   id,
			questions:   req.AskUser.Questions,
			questionIdx: 0,
			answers:     make([]any, 0, len(req.AskUser.Questions)),
		}
		ps.loadCurrentQuestion()
		return ps
	}
	return nil
}

func (p *Popup) loadCurrentQuestion() {
	q := p.questions[p.questionIdx]
	p.title = q.Question
	p.subtitle = q.Detail
	p.input = newPopupInput("", false)
	p.multi = nil

	switch {
	case len(q.Options) == 0 && q.Secret:
		p.kind = popupSecret
		p.maxVisible = 0
	case len(q.Options) == 0:
		p.kind = popupText
		p.maxVisible = 0
	case q.MultiSelect:
		p.kind = popupMultiSelect
		p.options = q.Options
		p.multi = make(map[int]bool, len(q.Options))
		p.maxVisible = cmdSelectorMaxVisible
	default:
		p.kind = popupSingleSelect
		p.options = q.Options
		p.maxVisible = cmdSelectorMaxVisible
	}
}

func (p *Popup) advanceOrResolve(answer any) (resolved bool, reply runtime.Reply) {
	p.answers = append(p.answers, answer)
	p.questionIdx++
	if p.questionIdx >= len(p.questions) {
		return true, runtime.Reply{Answers: p.answers}
	}
	p.loadCurrentQuestion()
	return false, runtime.Reply{}
}
