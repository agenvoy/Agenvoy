package msg

import "fmt"

type Label string

const (
	Invalid     Label = "Invalid"
	NotFound    Label = "Not found"
	Denied      Label = "Denied"
	Conflict    Label = "Conflict"
	Limit       Label = "Limit"
	Unavailable Label = "Unavailable"
	Timeout     Label = "Timeout"
	Upstream    Label = "Upstream"
	Canceled    Label = "Canceled"
	Internal    Label = "Internal"
)

func New(label Label, target, format string, args ...any) error {
	return fmt.Errorf("[%s] %s: "+format, append([]any{label, target}, args...)...)
}

func Wrap(label Label, fn, target string, err error) error {
	if target == "" {
		return fmt.Errorf("[%s] %s: %w", label, fn, err)
	}
	return fmt.Errorf("[%s] %s %s: %w", label, fn, target, err)
}

func Trace(fn, target string, err error) error {
	if target == "" {
		return fmt.Errorf("%s: %w", fn, err)
	}
	return fmt.Errorf("%s %s: %w", fn, target, err)
}
