package exec

import (
	"context"
	"errors"
	"strings"
)

func isSendTimeoutError(err error, sendCtxErr error) bool {
	if errors.Is(sendCtxErr, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	s := err.Error()

	if strings.Contains(s, "Client.Timeout") ||
		strings.Contains(s, "context deadline exceeded") ||
		strings.Contains(s, "timeout awaiting response headers") ||
		strings.Contains(s, "TLS handshake timeout") ||
		strings.Contains(s, "i/o timeout") {
		return true
	}
	return false
}
