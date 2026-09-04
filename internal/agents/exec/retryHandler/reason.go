package retryHandler

import (
	"strings"
	"time"
)

const (
	ReasonQuota       = "quota exhausted"
	ReasonRateLimited = "rate limited"
)

var retryIntervals = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	15 * time.Second,
}

func Handle(agentName string, err error, code, attempt int) (string, time.Duration) {
	reason := Reason(err, code)
	if reason == "" {
		return "", 0
	}
	Register(agentName)
	if reason != ReasonRateLimited || attempt >= len(retryIntervals) {
		return reason, 0
	}
	return reason, retryIntervals[attempt]
}

var quotaMarkers = []string{
	"spending-limit",
	"out of credits",
	"insufficient_quota",
	"insufficient quota",
	"usage_limit_reached",
	"quota exceeded",
	"billing",
}

var rateLimitMarkers = []string{
	"rate_limit_exceeded",
	"rate limit",
	"too many requests",
}

func Reason(err error, code int) string {
	if code == 402 || code == 403 {
		return ReasonQuota
	}

	if err != nil {
		str := strings.ToLower(err.Error())
		for _, marker := range quotaMarkers {
			if strings.Contains(str, marker) {
				return ReasonQuota
			}
		}
	}

	if code == 429 {
		return ReasonRateLimited
	}

	if err != nil {
		str := strings.ToLower(err.Error())
		for _, marker := range rateLimitMarkers {
			if strings.Contains(str, marker) {
				return ReasonRateLimited
			}
		}
	}
	return ""
}
