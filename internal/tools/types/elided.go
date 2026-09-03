package toolTypes

import "strings"

const Elided = "[elided]"

func IsElided(value string) bool {
	return strings.Contains(value, Elided)
}
