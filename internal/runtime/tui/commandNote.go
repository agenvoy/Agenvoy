package tui

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/knowledge"
	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

const noteTimeout = 10 * time.Second

type noteSpec struct {
	label  string
	list   func(ctx context.Context) ([]string, error)
	read   func(ctx context.Context, name string) (noteEntry, error)
	create func(ctx context.Context, name, content string) (string, error)
	update func(ctx context.Context, origin, rename, content string) (string, error)
}

var noteSpecs = map[string]noteSpec{
	"rule": {
		label:  "rule",
		list:   ruleList,
		read:   ruleRead,
		create: ruleCreate,
		update: ruleUpdate,
	},
	"knowledge": {
		label:  "knowledge",
		list:   knowledgeList,
		read:   knowledgeRead,
		create: knowledgeCreate,
		update: knowledgeUpdate,
	},
}

func knowledgeList(context.Context) ([]string, error) {
	records := knowledge.List()
	names := make([]string, 0, len(records))
	for _, one := range records {
		if one.Name != "" {
			names = append(names, one.Name)
		}
	}
	return names, nil
}

func knowledgeRead(_ context.Context, name string) (noteEntry, error) {
	record, ok := knowledge.Read(name)
	if !ok {
		return noteEntry{}, knowledge.ErrNotFound
	}
	return noteEntry{Name: record.Name, Content: record.Content}, nil
}

func knowledgeCreate(_ context.Context, name, content string) (string, error) {
	return knowledge.Create(name, content)
}

func knowledgeUpdate(_ context.Context, origin, rename, content string) (string, error) {
	return knowledge.Update(origin, rename, content)
}

func ruleList(ctx context.Context) ([]string, error) {
	out, err := daemon.Get[map[string][]noteEntry](ctx, "/v1/rules", nil)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out["rules"]))
	for _, one := range out["rules"] {
		if one.Name != "" {
			names = append(names, one.Name)
		}
	}
	return names, nil
}

func ruleRead(ctx context.Context, name string) (noteEntry, error) {
	return daemon.Get[noteEntry](ctx, "/v1/rule/"+url.PathEscape(name), nil)
}

func ruleCreate(ctx context.Context, name, content string) (string, error) {
	out, err := daemon.Post[noteEntry](ctx, "/v1/rule", map[string]any{"name": name, "content": content})
	return out.Name, err
}

func ruleUpdate(ctx context.Context, origin, rename, content string) (string, error) {
	body := map[string]any{"name": origin, "content": content}
	if rename != origin {
		body["rename"] = rename
	}
	out, err := daemon.Patch[noteEntry](ctx, "/v1/rule", body)
	return out.Name, err
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

		names, err := spec.list(ctx)
		if err != nil {
			return NoteListed{kind: kind, err: err}
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

		out, err := spec.read(ctx, name)
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

		var name string
		var err error
		if msg.origin == "" {
			name, err = spec.create(ctx, msg.title, msg.body)
		} else {
			name, err = spec.update(ctx, msg.origin, msg.title, msg.body)
		}
		if err != nil {
			return NoteSaved{kind: msg.kind, name: msg.title, err: err}
		}
		return NoteSaved{kind: msg.kind, name: name}
	}
}

func (t TUI) runNoteSaved(msg NoteSaved) (TUI, tea.Cmd) {
	spec := noteSpecs[msg.kind]
	if msg.err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s save %s: %v", spec.label, msg.name, msg.err)) + "\n")
	}
	return t, tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s saved: %s", spec.label, msg.name)) + "\n")
}
