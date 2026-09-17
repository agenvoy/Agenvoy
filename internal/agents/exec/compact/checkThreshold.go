package compact

import "strings"

const (
	thresholdRatio   = 0.8
	copilotWindow    = 256_000
	fallbackWindow   = 128_000
	copilotNamespace = "copilot@"
)

func CheckThreshold(modelName string) int {
	if in, ok := lookupLimit(modelName); ok {
		return int(float64(in) * thresholdRatio)
	}
	if strings.HasPrefix(strings.TrimSpace(modelName), copilotNamespace) {
		return int(copilotWindow * thresholdRatio)
	}
	return int(fallbackWindow * thresholdRatio)
}
