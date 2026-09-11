package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
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

	tags := make(map[string]string, len(cfg.ModelTag))
	for name, tag := range cfg.ModelTag {
		tags[NormalizeModel(name)] = tag
	}
	cfg.ModelTag = tags
}

const ModelTagPass = "pass"

var ModelTags = []string{"S", "A", "B", "C", ModelTagPass}

func ModelTagLines(cfg *Config) string {
	registered := make(map[string]bool, len(cfg.Models))
	for _, m := range cfg.Models {
		registered[m.Name] = true
	}
	groups := make(map[string][]string, len(ModelTags))
	for name, tag := range cfg.ModelTag {
		if registered[name] {
			groups[tag] = append(groups[tag], name)
		}
	}
	lines := make([]string, 0, len(ModelTags))
	for _, tag := range ModelTags {
		if names := groups[tag]; len(names) > 0 {
			slices.Sort(names)
			lines = append(lines, fmt.Sprintf("%q: %s", tag, strings.Join(names, ", ")))
		}
	}
	if len(lines) == 0 {
		return "(none set)"
	}
	return strings.Join(lines, "\n")
}

func SetModelTag(name, tag string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if tag != "" && !slices.Contains(ModelTags, tag) {
		return fmt.Errorf("unknown model tag %q", tag)
	}
	if tag == "" {
		delete(cfg.ModelTag, name)
	} else {
		cfg.ModelTag[name] = tag
	}
	return Save(cfg)
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
