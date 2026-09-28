package compact

import "strings"

const (
	thresholdRatio   = 0.8
	fallbackWindow   = 128_000
	copilotNamespace = "copilot@"
)

func CheckThreshold(modelName string) int {
	return int(float64(InputWindow(modelName)) * thresholdRatio)
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
	return fallbackWindow
}
