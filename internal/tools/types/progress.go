package toolTypes

import "context"

type progressKey struct{}

func WithProgress(ctx context.Context, send func(string)) context.Context {
	if send == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, send)
}

func Progress(ctx context.Context) func(string) {
	send, _ := ctx.Value(progressKey{}).(func(string))
	if send == nil {
		return func(string) {}
	}
	return send
}
