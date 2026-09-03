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

const noteTimeout = 10 * time.Second

type noteSpec struct {
	label    string
	listPath string
	listKey  string
	itemPath string
}

var noteSpecs = map[string]noteSpec{
	"rule":      {"rule", "/v1/rules", "rules", "/v1/rule"},
	"knowledge": {"knowledge", "/v1/knowledges", "knowledges", "/v1/knowledge"},
}

type noteEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type NoteListed struct {
	kind  string
	names []string
	err   error
}

type NotePick struct {
	kind string
	name string
}

type NoteLoaded struct {
	kind    string
	name    string
	content string
	err     error
}

type NoteTitleSubmit struct {
	kind   string
	origin string
	title  string
}

type NoteBodySubmit struct {
	kind   string
	origin string
	title  string
	body   string
}

type NoteSaved struct {
	kind string
	name string
	err  error
}

func (t TUI) commandNote(kind string) (TUI, tea.Cmd, bool) {
	spec, ok := noteSpecs[kind]
	if !ok {
		return t, nil, false
	}

	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), noteTimeout)
		defer cancel()

		out, err := daemon.Get[map[string][]noteEntry](ctx, spec.listPath, nil)
		if err != nil {
			return NoteListed{kind: kind, err: err}
		}

		names := make([]string, 0, len(out[spec.listKey]))
		for _, one := range out[spec.listKey] {
			if one.Name != "" {
				names = append(names, one.Name)
			}
		}
		sort.Strings(names)
		return NoteListed{kind: kind, names: names}
	}, true
}

func (t TUI) runNoteListed(msg NoteListed) (TUI, tea.Cmd) {
	spec := noteSpecs[msg.kind]
	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s list: %v", spec.label, msg.err)) + "\n")
	}

	kind := msg.kind
	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   fmt.Sprintf("%s · pick one to edit", spec.label),
		options: append([]string{"New"}, msg.names...),
		values:  append([]string{""}, msg.names...),
		onConfirm: func(chosen string) any {
			return NotePick{kind: kind, name: chosen}
		},
	}
	return t, nil
}

func (t TUI) runNotePick(msg NotePick) (TUI, tea.Cmd) {
	if msg.name == "" {
		t.noteBodyDraft = ""
		return t.showNoteTitlePopup(msg.kind, "", "")
	}

	spec := noteSpecs[msg.kind]
	kind, name := msg.kind, msg.name
	return t, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), noteTimeout)
		defer cancel()

		out, err := daemon.Get[noteEntry](ctx, spec.itemPath+"/"+url.PathEscape(name), nil)
		if err != nil {
			return NoteLoaded{kind: kind, name: name, err: err}
		}
		return NoteLoaded{kind: kind, name: out.Name, content: out.Content}
	}
}

func (t TUI) runNoteLoaded(msg NoteLoaded) (TUI, tea.Cmd) {
	spec := noteSpecs[msg.kind]
	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s read %s: %v", spec.label, msg.name, msg.err)) + "\n")
	}

	t.noteBodyDraft = msg.content
	return t.showNoteTitlePopup(msg.kind, msg.name, msg.name)
}

func (t TUI) showNoteTitlePopup(kind, origin, title string) (TUI, tea.Cmd) {
	spec := noteSpecs[kind]
	t.popup = &Popup{
		kind:  popupText,
		title: fmt.Sprintf("%s title", spec.label),
		input: newPopupInput(title, false),
		onConfirm: func(value string) any {
			return NoteTitleSubmit{kind: kind, origin: origin, title: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runNoteTitleSubmit(msg NoteTitleSubmit) (TUI, tea.Cmd) {
	spec := noteSpecs[msg.kind]
	if msg.title == "" {
		t.noteBodyDraft = ""
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s title required", spec.label)) + "\n")
	}
	return t.showNoteBodyPopup(msg.kind, msg.origin, msg.title)
}

func (t TUI) showNoteBodyPopup(kind, origin, title string) (TUI, tea.Cmd) {
	spec := noteSpecs[kind]
	t.popup = &Popup{
		kind:      popupText,
		title:     fmt.Sprintf("%s description (%s)", spec.label, title),
		multiline: true,
		input:     newPopupInput(t.noteBodyDraft, true),
		onConfirm: func(value string) any {
			return NoteBodySubmit{kind: kind, origin: origin, title: title, body: value}
		},
	}
	t.noteBodyDraft = ""
	return t, nil
}

func (t TUI) noteSaveCmd(msg NoteBodySubmit) tea.Cmd {
	spec := noteSpecs[msg.kind]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), noteTimeout)
		defer cancel()

		var out noteEntry
		var err error
		if msg.origin == "" {
			out, err = daemon.Post[noteEntry](ctx, spec.itemPath, map[string]any{
				"name":    msg.title,
				"content": msg.body,
			})
		} else {
			body := map[string]any{"name": msg.origin, "content": msg.body}
			if msg.title != msg.origin {
				body["rename"] = msg.title
			}
			out, err = daemon.Patch[noteEntry](ctx, spec.itemPath, body)
		}
		if err != nil {
			return NoteSaved{kind: msg.kind, name: msg.title, err: err}
		}
		return NoteSaved{kind: msg.kind, name: out.Name}
	}
}

func (t TUI) runNoteSaved(msg NoteSaved) (TUI, tea.Cmd) {
	spec := noteSpecs[msg.kind]
	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s save %s: %v", spec.label, msg.name, msg.err)) + "\n")
	}
	return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s saved: %s", spec.label, msg.name)) + "\n")
}
