package compact

import "strings"

// * templatily use, will be deprecated after I think about a better way
func CheckThreshold(modelName string) int {
	switch {
	case strings.Contains(modelName, "gemini"),
		strings.Contains(modelName, "gpt-5.4"),
		strings.Contains(modelName, "gpt-5.5"),
		strings.Contains(modelName, "gpt-5.6"):
		return int(1_000_000 * 0.8)
	case strings.Contains(modelName, "grok-4.5"):
		return int(500_000 * 0.8)
	case strings.Contains(modelName, "claude"):
		return int(200_000 * 0.8)
	case (strings.Contains(modelName, "gpt") && !strings.Contains(modelName, "gpt-oss")),
		strings.Contains(modelName, "grok"):
		return int(256_000 * 0.8)
	default:
		return int(128_000 * 0.8)
	}
}
