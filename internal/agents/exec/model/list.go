package model

import (
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func List() []agentTypes.ModelEntry {
	cfg, err := go_pkg_filesystem.ReadJSON[ConfigMeta](filesystem.ConfigPath)
	if err != nil || len(cfg.Models) == 0 {
		return []agentTypes.ModelEntry{}
	}
	for i := range cfg.Models {
		cfg.Models[i].Name = config.NormalizeModel(cfg.Models[i].Name)
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = cfg.Models[0].Name
	} else {
		for i, m := range cfg.Models {
			// * move default model to first be fallback
			if m.Name == cfg.DefaultModel {
				cfg.Models[0], cfg.Models[i] = cfg.Models[i], cfg.Models[0]
				break
			}
		}
	}
	return cfg.Models
}
