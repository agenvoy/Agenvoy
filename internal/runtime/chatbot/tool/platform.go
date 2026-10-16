package tool

import (
	"slices"
	"strings"

	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

const (
	platformTelegram = "telegram"
	platformDiscord  = "discord"
)

var platformEnum []string

func platformParam() map[string]any {
	return map[string]any{
		"type":        "string",
		"enum":        platformEnum,
		"description": "Target platform.",
	}
}

func parsePlatform(s string) (string, error) {
	if slices.Contains(platformEnum, s) {
		return s, nil
	}
	return "", utilsmsg.New(utilsmsg.Invalid, "platform", "unsupported %q; available: %s", s, strings.Join(platformEnum, ", "))
}
