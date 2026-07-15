package agentTypes

import "fmt"

type RateLimit struct {
	Agent    string
	ResetsAt int64
	Body     string
}

func (e *RateLimit) Error() string {
	return fmt.Sprintf("HTTP 429: rate limit until %d: %s", e.ResetsAt, e.Body)
}
