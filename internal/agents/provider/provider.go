package provider

import (
	"net/http"
	"strings"
	"time"
)

var reasoningLevel = "medium"

var reasoningLevelOrder = []string{"low", "medium", "high", "xhigh"}

func NormalizeReasoningLevel(s string) string {
	switch strings.ToLower(s) {
	case "low", "medium", "high", "xhigh":
		return strings.ToLower(s)
	default:
		return "medium"
	}
}

func SetReasoningLevel(level string) {
	reasoningLevel = NormalizeReasoningLevel(level)
}

func GetReasoningLevel() string {
	return reasoningLevel
}

func reasoningLevelIndex(level string) int {
	for i, v := range reasoningLevelOrder {
		if v == level {
			return i
		}
	}
	return 1
}

func ClampReasoningLevel(level, maxLevel string) string {
	if reasoningLevelIndex(level) > reasoningLevelIndex(maxLevel) {
		return maxLevel
	}
	return level
}

func MaxReasoningLevel(providerName, model string) string {
	switch providerName {
	case "openai", "codex", "copilot":
		if strings.Contains(model, "codex-max") {
			return "xhigh"
		}
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

func ResponsesAPI(providerName, model string) bool {
	switch providerName {
	case "openai":
		return strings.Contains(model, "codex") || strings.HasPrefix(model, "gpt-5.4") || strings.HasSuffix(model, "-pro")
	case "copilot":
		return strings.Contains(model, "-codex") || strings.HasPrefix(model, "gpt-5.4")
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

func ThinkingBudget(level string) int {
	switch level {
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
