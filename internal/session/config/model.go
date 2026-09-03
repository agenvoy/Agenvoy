package config

import (
	"strings"
)

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
	cfg, err := Load()
	if err != nil {
		return ""
	}

	for _, v := range cfg.Compats {
		if strings.EqualFold(v.Provider, provider) {
			return v.URL
		}
	}
	return ""
}
