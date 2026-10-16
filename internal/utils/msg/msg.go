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

func New(label Label, target, reason string) error {
	return fmt.Errorf("[%s] %s: %s", label, target, reason)
}

func Wrap(label Label, fn, target string, err error) error {
	if target == "" {
		return fmt.Errorf("[%s] %s: %w", label, fn, err)
	}
	return fmt.Errorf("[%s] %s %s: %w", label, fn, target, err)
}
