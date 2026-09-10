package tui

import (
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func registeredModelLines() []string {
	cfg, err := config.Load()
	if err != nil || cfg == nil || len(cfg.Models) == 0 {
		return []string{hintStyle.Render("  no models configured")}
	}

	lines := make([]string, 0, len(cfg.Models)+1)
	for _, m := range cfg.Models {
		label := hintStyle.Render("  " + m.Name)
		if cfg.DispatcherModel != "" && m.Name == cfg.DispatcherModel {
			label += "  " + systemStyle.Render("[dispatcher]")
		}
		if cfg.SummaryModel != "" && m.Name == cfg.SummaryModel {
			label += "  " + systemStyle.Render("[summary]")
		}
		lines = append(lines, label)
	}
	return lines
}
