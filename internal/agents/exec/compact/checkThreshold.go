package compact

import (
	"strings"

	"github.com/pardnchiu/agenvoy/configs"
)

const (
	copilotNamespace = "copilot@"
)

func CheckThreshold(modelName string) int {
	return int(float64(InputWindow(modelName)) * configs.COMPACT_THRESHOLD_RATIO)
}

func InputWindow(modelName string) int {
	if in, ok := lookupLimit(modelName); ok {
		return in
	}
	if model, ok := strings.CutPrefix(strings.TrimSpace(modelName), copilotNamespace); ok {
		WarmCopilot()
		if in, ok := copilotLimit(model); ok {
			return in
		}
	}
	return configs.FALLBACK_CONTEXT_WINDOW
}
