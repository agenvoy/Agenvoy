package tui

import (
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

type Pending struct {
	id           string
	request      runtime.Request
	needPassword bool
}

func resolvePending(id string, reply runtime.Reply) {
	resolvePendingWithPassword(id, reply, "")
}

func resolvePendingWithPassword(id string, reply runtime.Reply, password string) {
	ipcClient.Load().Reply(id, reply, password)
}
