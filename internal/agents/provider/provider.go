package provider

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

var reasoningLevelOrder = []string{"none", "low", "medium", "high", "xhigh"}

const RateLimitCooldown = 30 * time.Minute

func ReasoningDisabled(level string) bool {
	return level == "none"
}

func reasoningLevelIndex(level string) int {
	for i, v := range reasoningLevelOrder {
		if v == level {
			return i
		}
	}
	return 2
}

func ClampReasoningLevel(level, maxLevel string) string {
	if reasoningLevelIndex(level) > reasoningLevelIndex(maxLevel) {
		return maxLevel
	}
	return level
}

func MinReasoningLevel(providerName, model string) string {
	if providerName == "gemini" && GetThinkingConfig(providerName, model) == "level" {
		return "low"
	}
	return "none"
}

func FloorReasoningLevel(level, minLevel string) string {
	if reasoningLevelIndex(level) < reasoningLevelIndex(minLevel) {
		return minLevel
	}
	return level
}

func MaxReasoningLevel(providerName, model string) string {
	switch providerName {
	case "openai", "codex":
		if strings.Contains(model, "codex-max") || strings.Contains(model, "gpt-5.6") {
			return "xhigh"
		}
		return "high"
	case "copilot":
		return "high"
	case "openrouter":
		return "xhigh"
	case "claude":
		if strings.Contains(model, "-20") {
			return "high"
		}
		if strings.Contains(model, "opus-4-7") || strings.Contains(model, "opus-4-8") ||
			strings.Contains(model, "fable-5") || strings.Contains(model, "mythos-5") {
			return "xhigh"
		}
		return "high"
	}
	return "high"
}

func SupportTemperature(providerName, model string) bool {
	switch providerName {
	case "openai", "copilot", "codex":
		if strings.HasPrefix(model, "gpt-5") {
			return false
		}
	case "deepseek":
		if model == "deepseek-reasoner" {
			return false
		}
	case "claude":
		return false
	case "gemini":
		if strings.Contains(model, "-preview") {
			return false
		}
	}
	return true
}

func gpt5MinorAtLeast(model string, min int) bool {
	rest, ok := strings.CutPrefix(model, "gpt-5.")
	if !ok {
		return false
	}
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if end == -1 {
		end = len(rest)
	}
	n, err := strconv.Atoi(rest[:end])
	return err == nil && n >= min
}

func ResponsesAPI(providerName, model string) bool {
	switch providerName {
	case "openai":
		return strings.Contains(model, "codex") || gpt5MinorAtLeast(model, 4) || strings.HasSuffix(model, "-pro")
	case "copilot":
		return strings.Contains(model, "-codex") || gpt5MinorAtLeast(model, 4)
	}
	return false
}

func SupportReasoningEffort(providerName, model string) bool {
	switch providerName {
	case "openai", "copilot":
		if !strings.HasPrefix(model, "gpt-5") {
			return false
		}
		if strings.Contains(model, "-codex") || strings.HasSuffix(model, "-pro") {
			return false
		}
		return true
	case "grok", "grok-oauth":
		return !strings.Contains(model, "non-reasoning")
	}
	return false
}

func SupportsReasoningSwitch(providerName, model string) bool {
	switch providerName {
	case "nvidia", "deepseek", "cloudflare", "compat":
		return false
	case "codex":
		return true
	case "openai", "copilot":
		return ResponsesAPI(providerName, model) || SupportReasoningEffort(providerName, model)
	case "grok", "grok-oauth":
		return SupportReasoningEffort(providerName, model)
	case "claude":
		return GetThinkingType(providerName, model) != ""
	case "gemini":
		return GetThinkingConfig(providerName, model) != ""
	case "openrouter":
		vendor, _, _ := strings.Cut(model, "/")
		return vendor != "deepseek"
	}
	return true
}

func GetThinkingType(providerName, model string) string {
	if providerName != "claude" {
		return ""
	}
	if strings.Contains(model, "-20") {
		return "enabled"
	}
	return "adaptive"
}

func GetThinkingConfig(providerName, model string) string {
	if providerName != "gemini" {
		return ""
	}
	if strings.HasPrefix(model, "gemini-2.5-") {
		return "budget"
	}
	if strings.HasPrefix(model, "gemini-3") {
		return "level"
	}
	return ""
}

func ThinkingBudget(model, level string) int {
	switch level {
	case "none":
		if strings.Contains(model, "2.5-pro") {
			return 128
		}
		return 0
	case "low":
		return 1024
	case "high":
		return 16384
	default:
		return 8192
	}
}

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Minute}
}
