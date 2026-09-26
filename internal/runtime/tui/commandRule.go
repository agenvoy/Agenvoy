package tui

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

const ruleTimeout = 10 * time.Second

type ruleEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type RuleListed struct {
	names []string
	err   error
}

type RulePick struct {
	name string
}

type RuleLoaded struct {
	name    string
	content string
	err     error
}

type RuleTitleSubmit struct {
	origin string
	title  string
}

type RuleBodySubmit struct {
	origin string
	title  string
	body   string
}

type RuleSaved struct {
	name string
	err  error
}

func (t TUI) commandRule() (TUI, tea.Cmd, bool) {
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ruleTimeout)
		defer cancel()

		out, err := daemon.Get[map[string][]ruleEntry](ctx, "/v1/rules", nil)
		if err != nil {
			return RuleListed{err: err}
		}
		names := make([]string, 0, len(out["rules"]))
		for _, one := range out["rules"] {
			if one.Name != "" {
				names = append(names, one.Name)
			}
		}
		sort.Strings(names)
		return RuleListed{names: names}
	}, true
}

func (t TUI) runRuleListed(msg RuleListed) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("rule list: %v", msg.err)) + "\n")
	}

	options := []string{"New"}
	values := []string{""}
	if len(msg.names) > 0 {
		options = append(options, "")
		values = append(values, "")
	}

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "/rule",
		subtitle: "pick one to edit",
		options:  append(options, msg.names...),
		values:   append(values, msg.names...),
		onConfirm: func(chosen string) any {
			return RulePick{name: chosen}
		},
	}
	return t, nil
}

func (t TUI) runRulePick(msg RulePick) (TUI, tea.Cmd) {
	if msg.name == "" {
		t.ruleBodyDraft = ""
		return t.showRuleTitlePopup("", "")
	}

	name := msg.name
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ruleTimeout)
		defer cancel()

		out, err := daemon.Get[ruleEntry](ctx, "/v1/rule/"+url.PathEscape(name), nil)
		if err != nil {
			return RuleLoaded{name: name, err: err}
		}
		return RuleLoaded{name: out.Name, content: out.Content}
	}
}

func (t TUI) runRuleLoaded(msg RuleLoaded) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("rule read %s: %v", msg.name, msg.err)) + "\n")
	}

	t.ruleBodyDraft = msg.content
	return t.showRuleTitlePopup(msg.name, msg.name)
}

func (t TUI) showRuleTitlePopup(origin, title string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:  popupText,
		title: "rule title",
		input: newPopupInput(title, false),
		onConfirm: func(value string) any {
			return RuleTitleSubmit{origin: origin, title: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runRuleTitleSubmit(msg RuleTitleSubmit) (TUI, tea.Cmd) {
	if msg.title == "" {
		t.ruleBodyDraft = ""
		return t, tea.Println(msgError("rule title required") + "\n")
	}
	return t.showRuleBodyPopup(msg.origin, msg.title)
}

func (t TUI) showRuleBodyPopup(origin, title string) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:      popupText,
		title:     fmt.Sprintf("rule description (%s)", title),
		multiline: true,
		input:     newPopupInput(t.ruleBodyDraft, true),
		onConfirm: func(value string) any {
			return RuleBodySubmit{origin: origin, title: title, body: value}
		},
	}
	t.ruleBodyDraft = ""
	return t, nil
}

func (t TUI) ruleSaveCmd(msg RuleBodySubmit) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ruleTimeout)
		defer cancel()

		var out ruleEntry
		var err error
		if msg.origin == "" {
			out, err = daemon.Post[ruleEntry](ctx, "/v1/rule", map[string]any{"name": msg.title, "content": msg.body})
		} else {
			body := map[string]any{"name": msg.origin, "content": msg.body}
			if msg.title != msg.origin {
				body["rename"] = msg.title
			}
			out, err = daemon.Patch[ruleEntry](ctx, "/v1/rule", body)
		}
		if err != nil {
			return RuleSaved{name: msg.title, err: err}
		}
		return RuleSaved{name: out.Name}
	}
}

func (t TUI) runRuleSaved(msg RuleSaved) (TUI, tea.Cmd) {
	if msg.err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("rule save %s: %v", msg.name, msg.err)) + "\n")
	}
	return t, tea.Println(msgLog(fmt.Sprintf("rule saved: %s", msg.name)) + "\n")
}
