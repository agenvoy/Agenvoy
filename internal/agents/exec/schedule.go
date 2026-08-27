package exec

import "context"

type scheduleKey struct{}

func WithSchedule(ctx context.Context) context.Context {
	return context.WithValue(ctx, scheduleKey{}, true)
}

func isSchedule(ctx context.Context) bool {
	value, _ := ctx.Value(scheduleKey{}).(bool)
	return value
}

func scheduleLabel(name string) string {
	if name == "" {
		return "Schedule"
	}
	return "Schedule[" + name + "]"
}
