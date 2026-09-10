package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
)

type PendingSelect struct {
	id       string
	taskHash string
}

type PendingDeletePick struct {
	id       string
	taskHash string
	label    string
}

type PendingDeleteConfirm struct {
	id       string
	taskHash string
	label    string
	yes      bool
}

func (t TUI) commandPending() (TUI, tea.Cmd, bool) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(msgLog("no active session") + "\n"), true
	}

	hashes := interactive.ListResumablePending(sid)
	if len(hashes) == 0 {
		return t, tea.Println(msgLog("no pending tasks") + "\n"), true
	}

	options := make([]string, 0, len(hashes))
	values := make([]string, 0, len(hashes))
	for _, h := range hashes {
		info, ok := interactive.LoadPendingInfo(sid, h)
		if !ok {
			continue
		}
		label := go_pkg_utils.TruncateString(h, 8)
		if info.Objective != "" {
			label = go_pkg_utils.TruncateString(info.Objective, 64)
		}
		if !info.UpdatedAt.IsZero() {
			label = info.UpdatedAt.Local().Format("01-02 15:04") + "  " + label
		}
		if info.HasQuestions {
			label += " (awaiting answer)"
		}
		options = append(options, label)
		values = append(values, h)
	}

	if len(options) == 0 {
		return t, tea.Println(msgLog("no pending tasks") + "\n"), true
	}

	sessionID := sid
	labels := make(map[string]string, len(values))
	for i, h := range values {
		labels[h] = options[i]
	}

	t.popup = &Popup{
		kind:       popupSingleSelect,
		title:      fmt.Sprintf("Pending tasks (%d)", len(options)),
		options:    options,
		values:     values,
		maxVisible: cmdSelectorMaxVisible,
		onConfirm: func(chosen string) any {
			return PendingSelect{id: sessionID, taskHash: chosen}
		},
		onDelete: func(chosen string) any {
			return PendingDeletePick{id: sessionID, taskHash: chosen, label: labels[chosen]}
		},
	}
	return t, nil, true
}

func (t TUI) openPendingDeleteConfirm(msg PendingDeletePick) (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Drop pending task ?",
		subtitle: msg.label,
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		onConfirm: func(chosen string) any {
			return PendingDeleteConfirm{id: msg.id, taskHash: msg.taskHash, label: msg.label, yes: chosen == "yes"}
		},
		onCancel: func() any {
			return PendingDeleteConfirm{id: msg.id, taskHash: msg.taskHash, label: msg.label}
		},
	}
	return t, nil
}

func (t TUI) runPendingDelete(msg PendingDeleteConfirm) (TUI, tea.Cmd) {
	if _, ok := interactive.LoadPendingInfo(msg.id, msg.taskHash); !ok {
		next, cmd, _ := t.commandPending()
		return next, tea.Sequence(tea.Println(msgLog("pending task already resolved in another session")+"\n"), cmd)
	}

	interactive.DeletePending(msg.id, msg.taskHash)

	next, cmd, _ := t.commandPending()
	return next, tea.Sequence(tea.Println(msgLog("dropped pending task: "+msg.label)+"\n"), cmd)
}

func (t TUI) resumePending(msg PendingSelect) (tea.Model, tea.Cmd) {
	info, ok := interactive.LoadPendingInfo(msg.id, msg.taskHash)
	if !ok {
		return t, tea.Println(msgLog("pending task already resolved in another session") + "\n")
	}

	if !info.HasQuestions {
		allowAll := interactive.LoadPendingAllowAll(msg.id, msg.taskHash)
		full, history, err := interactive.LoadResumeMessage(msg.id, msg.taskHash, nil)
		if err != nil {
			return t, tea.Println(msgError(fmt.Sprintf("load resume: %v", err)) + "\n")
		}
		return t.startResume(ResumeExec{SessionID: msg.id, Content: full, PendingTask: msg.taskHash, HistoryContent: history, AllowAll: allowAll})
	}

	meta, err := interactive.LoadPendingQuestions(msg.id, msg.taskHash)
	if err != nil {
		return t, tea.Println(msgError(fmt.Sprintf("load pending: %v", err)) + "\n")
	}

	sid := msg.id
	taskHash := msg.taskHash
	runtime.AskUser(runtime.Request{
		Kind:      runtime.KindAskUser,
		SessionID: sid,
		ToolName:  "ask_user",
		AskUser:   &runtime.UserPayload{Questions: meta},
	}, func(reply runtime.Reply) {
		if reply.Error != nil {
			interactive.CleanupPending(sid, taskHash)
			return
		}
		runtime.TriggerResume(sid, taskHash, reply.Answers)
	})
	return t, nil
}
