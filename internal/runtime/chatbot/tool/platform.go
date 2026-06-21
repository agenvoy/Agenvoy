package tool

import (
	"fmt"
	"slices"
	"strings"
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
	return "", fmt.Errorf("unsupported platform %q; available: %s", s, strings.Join(platformEnum, ", "))
}
