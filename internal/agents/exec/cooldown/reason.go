package cooldown

import "strings"

var quotaMarkers = []string{
	"spending-limit",
	"out of credits",
	"insufficient_quota",
	"insufficient quota",
	"usage_limit_reached",
	"quota exceeded",
	"billing",
}

func Reason(err error, code int) string {
	if code == 402 || code == 403 {
		return "quota exhausted"
	}

	if err != nil {
		str := strings.ToLower(err.Error())
		for _, marker := range quotaMarkers {
			if strings.Contains(str, marker) {
				return "quota exhausted"
			}
		}
	}

	if code == 429 {
		return "rate limited"
	}
	return ""
}
