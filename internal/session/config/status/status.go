package configStatus

import (
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
)

func Get(sessionID string) Status {
	if sessionID == "" {
		return Status{}
	}
	return FromCount(interactive.ActiveCount(sessionID))
}

func FromCount(count int) Status {
	if count <= 0 {
		return Status{State: StatusIdle}
	}
	return Status{State: StatusOnline, Count: count}
}
