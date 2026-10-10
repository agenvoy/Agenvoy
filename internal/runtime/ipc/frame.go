package ipc

import (
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

const (
	FrameRun   = "run"
	FrameEvent = "event"
	FrameDone  = "done"
)

type Frame struct {
	Type      string            `json:"type"`
	SessionID string            `json:"session_id,omitempty"`
	Error     string            `json:"error,omitempty"`
	Rayload   *Payload          `json:"run,omitempty"`
	Event     *agentTypes.Event `json:"event,omitempty"`
}

type Payload struct {
	Input          string `json:"input"`
	Model          string `json:"model,omitempty"`
	Reasoning      string `json:"reasoning,omitempty"`
	WorkDir        string `json:"work_dir"`
	AllowAll       bool   `json:"allow_all,omitempty"`
	PendingTask    string `json:"pending_task,omitempty"`
	HistoryContent string `json:"history_content,omitempty"`
}
