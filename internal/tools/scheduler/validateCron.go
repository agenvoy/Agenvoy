package scheduler

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

var cronDescriptors = []string{"@yearly", "@annually", "@monthly", "@weekly", "@daily", "@midnight", "@hourly"}

func ValidateCron(expression string) error {
	expression = strings.TrimSpace(expression)

	if !strings.HasPrefix(expression, "@") {
		if len(strings.Fields(expression)) != 5 {
			return fmt.Errorf("expression must be 5 fields '{min} {hour} {dom} {mon} {dow}', a descriptor (%s) or '@every {duration}' (got %q)", strings.Join(cronDescriptors, " "), expression)
		}
		return nil
	}

	if slices.Contains(cronDescriptors, expression) {
		return nil
	}

	rest, ok := strings.CutPrefix(expression, "@every ")
	if !ok {
		return fmt.Errorf("unknown descriptor %q; use one of %s or '@every {duration}'", expression, strings.Join(cronDescriptors, " "))
	}

	duration, err := time.ParseDuration(strings.TrimSpace(rest))
	if err != nil {
		return fmt.Errorf("invalid '@every' duration in %q: %w", expression, err)
	}
	if duration < 30*time.Second {
		return fmt.Errorf("'@every' minimum interval is 30s (got %q)", expression)
	}
	return nil
}
