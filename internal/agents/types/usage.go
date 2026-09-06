package agentTypes

import (
	"fmt"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func InputTotals(usage *provider.Usage) (total, hitPct int) {
	if usage == nil {
		return 0, 0
	}
	total = usage.Input + usage.CacheRead + usage.CacheCreate
	if usage.CacheRead > 0 && total > 0 {
		hitPct = int(float64(usage.CacheRead) / float64(total) * 100)
	}
	return total, hitPct
}

func FormatInput(total, hitPct int) string {
	if total <= 0 {
		return ""
	}
	if hitPct > 0 {
		return fmt.Sprintf("%s(%d%%)", go_pkg_utils.CompactNumber(total), hitPct)
	}
	return go_pkg_utils.CompactNumber(total)
}
