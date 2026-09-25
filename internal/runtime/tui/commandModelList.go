package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

const sessionModelPrefix = "model:"

type SessionModelSelect struct {
	name string
}

func registeredModelOptions(sid string) (options, values []string, cursor int) {
	cfg, err := config.Load()
	if err != nil || cfg == nil || len(cfg.Models) == 0 {
		return nil, nil, 0
	}

	current := ""
	if sid != "" {
		current, _ = configBot.GetModel(sid)
	}

	options = make([]string, 0, len(cfg.Models)+1)
	values = make([]string, 0, len(cfg.Models)+1)

	auto := configBot.DefaultModel
	if current == configBot.DefaultModel {
		auto += "  " + systemStyle.Render("[current]")
	}
	options = append(options, auto)
	values = append(values, sessionModelPrefix+configBot.DefaultModel)

	for _, m := range cfg.Models {
		label := "⇅ " + m.Name
		if m.Name == current {
			label += "  " + systemStyle.Render("[current]")
			cursor = len(options)
		}
		if cfg.DispatcherModel != "" && m.Name == cfg.DispatcherModel {
			label += "  " + okayStyle.Render("[dispatcher]")
		}
		if cfg.SummaryModel != "" && m.Name == cfg.SummaryModel {
			label += "  " + okayStyle.Render("[summary]")
		}
		switch tag := cfg.ModelTag[m.Name]; tag {
		case "":
		case config.ModelTagPass:
			label += "  " + warnStyle.Render(tag)
		default:
			label += "  " + warnStyle.Render(tag+"-tier")
		}
		options = append(options, label)
		values = append(values, sessionModelPrefix+m.Name)
	}
	return options, values, cursor
}

func (t TUI) runSessionModelSelect(name string) (TUI, tea.Cmd) {
	sid := strings.TrimSpace(t.currentSessionID)
	if sid == "" {
		return t, tea.Println(msgLog("no active session") + "\n")
	}
	configBot.SetModel(sid, name, "")
	return t, nil
}

func swapModelPriority(name, other string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	i := slices.IndexFunc(cfg.Models, func(m config.ModelEntry) bool { return m.Name == name })
	j := slices.IndexFunc(cfg.Models, func(m config.ModelEntry) bool { return m.Name == other })
	if i == -1 || j == -1 {
		return fmt.Errorf("model not registered: %s / %s", name, other)
	}
	cfg.Models[i], cfg.Models[j] = cfg.Models[j], cfg.Models[i]
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	agents.Reload()
	return nil
}

func providerPopup(title string, available []string, current string, back *Popup, onConfirm func(chosen string) any) *Popup {
	popup := &Popup{
		kind:      popupSingleSelect,
		title:     title,
		back:      back,
		tabs:      providerTabs(available),
		onConfirm: onConfirm,
	}
	popup.onTab = func(p *Popup) {
		fillProviderOptions(p, available, current)
	}
	popup.onTab(popup)
	return popup
}

func modelProvider(name string) string {
	provider, _, _ := strings.Cut(name, "@")
	return provider
}

func providerTabs(available []string) []string {
	var providers []string
	for _, name := range available {
		if provider := modelProvider(name); !slices.Contains(providers, provider) {
			providers = append(providers, provider)
		}
	}
	if len(providers) < 2 {
		return nil
	}
	return append([]string{"all"}, providers...)
}

func fillProviderOptions(p *Popup, available []string, current string) {
	tab := ""
	if p.tabIdx > 0 && p.tabIdx < len(p.tabs) {
		tab = p.tabs[p.tabIdx]
	}

	disable := "disable"
	if current == "" || current == "off" {
		disable += "  " + systemStyle.Render("[current]")
	}
	options := make([]string, 0, len(available)+1)
	values := make([]string, 0, len(available)+1)
	options = append(options, disable)
	values = append(values, "")
	cursor := 0
	for _, name := range available {
		if tab != "" && modelProvider(name) != tab {
			continue
		}
		label := name
		if current == name {
			label += "  " + systemStyle.Render("[current]")
			cursor = len(options)
		}
		options = append(options, label)
		values = append(values, name)
	}

	p.options = options
	p.values = values
	p.cursor = cursor
}
