package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

var remoteSkills = []string{
	"code-reviewer",
	"commit-generate",
	"readme-generate",
	"seo-optimize",
	"version-generate",
	"wiki-generate",
}

type SkillsInstallPick struct {
	names []string
}

type SkillsInstallDone struct {
	installed []string
	removed   []string
	failed    []string
}

func (t TUI) commandSkills() (TUI, tea.Cmd, bool) {
	multi := make(map[int]bool, len(remoteSkills))
	for i, name := range remoteSkills {
		if go_pkg_filesystem_reader.Exists(filepath.Join(filesystem.SystemDesignDir, name)) {
			multi[i] = true
		}
	}

	t.popup = &Popup{
		kind:     popupMultiSelect,
		title:    "Skills",
		subtitle: "checked: git clone github.com/agenvoy/skill-<name>  unchecked: remove  " + filesystem.SystemDesignDir,
		options:  slices.Clone(remoteSkills),
		values:   remoteSkills,
		multi:    multi,
		onConfirm: func(chosen string) any {
			if chosen == "" {
				return SkillsInstallPick{}
			}
			return SkillsInstallPick{names: strings.Split(chosen, "\x1F")}
		},
	}
	return t, nil, true
}

func (t TUI) runSkillsInstallPick(msg SkillsInstallPick) (TUI, tea.Cmd) {
	names := msg.names
	return t, func() tea.Msg {
		return syncSkills(names)
	}
}

func syncSkills(selected []string) SkillsInstallDone {
	var done SkillsInstallDone
	if err := go_pkg_filesystem.CheckDir(filesystem.SystemDesignDir, true); err != nil {
		done.failed = append(done.failed, fmt.Sprintf("%s: %v", filesystem.SystemDesignDir, err))
		return done
	}

	for _, name := range remoteSkills {
		dir := filepath.Join(filesystem.SystemDesignDir, name)
		exists := go_pkg_filesystem_reader.Exists(dir)
		want := slices.Contains(selected, name)

		switch {
		case want && !exists:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			out, err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "https://github.com/agenvoy/skill-"+name+".git", dir).CombinedOutput()
			cancel()
			if err != nil {
				done.failed = append(done.failed, fmt.Sprintf("clone %s: %v %s", name, err, strings.TrimSpace(string(out))))
				continue
			}
			done.installed = append(done.installed, name)

		case !want && exists:
			if err := os.RemoveAll(dir); err != nil {
				done.failed = append(done.failed, fmt.Sprintf("remove %s: %v", name, err))
				continue
			}
			done.removed = append(done.removed, name)
		}
	}

	if scanner := agents.Scanner(); scanner != nil {
		scanner.Scan()
	}
	return done
}

func (t TUI) runSkillsInstallDone(msg SkillsInstallDone) (TUI, tea.Cmd) {
	var cmds []tea.Cmd
	if len(msg.installed) > 0 {
		cmds = append(cmds, tea.Println(msgLog("installed: "+strings.Join(msg.installed, ", "))+"\n"))
	}
	if len(msg.removed) > 0 {
		cmds = append(cmds, tea.Println(msgLog("removed: "+strings.Join(msg.removed, ", "))+"\n"))
	}
	for _, line := range msg.failed {
		cmds = append(cmds, tea.Println(msgError(line)+"\n"))
	}
	if len(cmds) == 0 {
		return t, tea.Println(msgLog("skills unchanged") + "\n")
	}
	return t, tea.Sequence(cmds...)
}
