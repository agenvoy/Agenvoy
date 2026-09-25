package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents"
	allowSkill "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/skill"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

var remoteSkills = []string{
	"code-reviewer",
	"commit-generate",
	"go-test-generate",
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
	installMulti := make(map[int]bool, len(remoteSkills))
	for i, name := range remoteSkills {
		if go_pkg_filesystem_reader.Exists(filepath.Join(filesystem.SystemDesignDir, name)) {
			installMulti[i] = true
		}
	}

	popup := &Popup{
		title: "/skill",
		tabs:  []string{"permission", "system"},
		onConfirm: func(chosen string) any {
			if chosen == "" {
				return SkillsInstallPick{}
			}
			return SkillsInstallPick{names: strings.Split(chosen, "\x1F")}
		},
		openLabel: "folder",
		onOpen: func(chosen string) string {
			scanner := agents.Scanner()
			if scanner == nil {
				return ""
			}
			one := scanner.Lookup(chosen)
			if one == nil {
				return ""
			}
			return filepath.Dir(one.AbsPath)
		},
		onEnter: func(p *Popup, chosen string) tea.Cmd {
			if _, err := allowSkill.ToggleGlobal(chosen); err != nil {
				p.subtitle = errorStyle.Render(fmt.Sprintf("allow %s: %v", chosen, err))
				return nil
			}
			fillAllowSkills(p)
			return nil
		},
	}
	popup.onTab = func(p *Popup) {
		p.cursor = 0
		if p.tabIdx == 1 {
			p.kind = popupMultiSelect
			p.enterAction = ""
			p.subtitle = "checked: git clone github.com/agenvoy/skill-<name>  unchecked: remove  " + filesystem.SystemDesignDir
			p.options = slices.Clone(remoteSkills)
			p.values = remoteSkills
			p.multi = installMulti
			return
		}
		p.kind = popupSingleSelect
		p.enterAction = "toggle"
		fillAllowSkills(p)
	}
	popup.onTab(popup)
	t.popup = popup
	return t, nil, true
}

func fillAllowSkills(p *Popup) {
	var names []string
	scanner := agents.Scanner()
	if scanner != nil {
		names = scanner.List()
		sort.Strings(names)
	}
	allowed := allowSkill.LoadGlobal()
	sources := make([]string, len(names))
	sourceWidth := 0
	for i, name := range names {
		if one := scanner.Lookup(name); one != nil {
			sources[i] = runtime.SkillSource(one.AbsPath)
		}
		sourceWidth = max(sourceWidth, len(sources[i])+2)
	}
	settings := make([]string, len(names))
	for i, name := range names {
		status := hintStyle.Render("ask")
		if allowed[name] {
			status = okayStyle.Render("always allow")
		}
		settings[i] = hintStyle.Render(padToWidth(sources[i], sourceWidth)) + status
	}
	p.subtitle = "always allowed skills skip the permission prompt  " + filesystem.AllowSkillGlobalPath
	p.options = optionColumn(names, settings)
	p.values = names
	p.cursor = min(p.cursor, max(len(p.options)-1, 0))
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
