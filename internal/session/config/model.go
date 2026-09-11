package config

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
)

type LocalCompat struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
	URL      string `json:"url"`
}

var LocalCompats = loadLocalCompats()

func loadLocalCompats() []LocalCompat {
	var list []LocalCompat
	if err := json.Unmarshal(configs.LocalCompat, &list); err != nil {
		slog.Warn("embedded local_compat",
			slog.String("error", err.Error()))
	}
	return list
}

func NormalizeModel(name string) string {
	head, model, ok := strings.Cut(name, "@")
	if !ok {
		return name
	}
	instance, found := strings.CutPrefix(head, "compat[")
	if !found {
		return name
	}
	instance, closed := strings.CutSuffix(instance, "]")
	if !closed || instance == "" {
		return name
	}
	return strings.ToLower(instance) + "@" + model
}

func normalizeModels(cfg *Config) {
	cfg.DispatcherModel = NormalizeModel(cfg.DispatcherModel)
	cfg.SummaryModel = NormalizeModel(cfg.SummaryModel)

	seen := make(map[string]bool, len(cfg.Models))
	kept := cfg.Models[:0]
	for _, m := range cfg.Models {
		m.Name = NormalizeModel(m.Name)
		if seen[m.Name] {
			continue
		}
		seen[m.Name] = true
		kept = append(kept, m)
	}
	cfg.Models = kept
}

func UpsertCompat(provider, url string) error {
	provider = strings.ToUpper(strings.TrimSpace(provider))
	cfg, err := Load()
	if err != nil {
		return err
	}

	for k, v := range cfg.Compats {
		if strings.EqualFold(v.Provider, provider) {
			cfg.Compats[k].URL = url
			return Save(cfg)
		}
	}

	cfg.Compats = append(cfg.Compats, CompatEntry{
		Provider: provider,
		URL:      url,
	})
	return Save(cfg)
}

func GetCompatURL(provider string) string {
	if cfg, err := Load(); err == nil {
		for _, v := range cfg.Compats {
			if strings.EqualFold(v.Provider, provider) {
				return v.URL
			}
		}
	}

	for _, v := range LocalCompats {
		if strings.EqualFold(v.Provider, provider) {
			return v.URL
		}
	}
	return ""
}
