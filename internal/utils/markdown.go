package utils

import "strings"

type FenceState struct {
	InFence bool
}

func (f *FenceState) Normalize(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")

	marker := strings.HasPrefix(trimmed, "```")
	if marker {
		f.InFence = !f.InFence
	}
	if f.InFence {
		return line, marker
	}
	return trimmed, marker
}
