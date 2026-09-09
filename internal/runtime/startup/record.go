package startup

import (
	"log/slog"

	"github.com/pardnchiu/agenvoy/internal/session/config"
)

const configKey = "startup"

func record(enabled bool) {
	dic, err := config.Get()
	if err != nil {
		dic = map[string]any{}
	}
	dic[configKey] = enabled
	if err := config.Write(dic); err != nil {
		slog.Warn("startup: recording the setting failed",
			slog.Bool("enabled", enabled),
			slog.String("error", err.Error()))
	}
}

func Recorded() (bool, bool) {
	dic, err := config.Get()
	if err != nil {
		return false, false
	}
	value, ok := dic[configKey].(bool)
	return value, ok
}
