package ipc

import (
	"context"
	"errors"
	"strings"

	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/guide"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func (c *conn) run(ctx context.Context, f Frame) {
	if f.Rayload == nil {
		c.write(Frame{Type: FrameDone, UUID: f.UUID, Error: "run payload is required"})
		return
	}

	sessionID := strings.TrimSpace(f.SessionID)
	if sessionID == "" {
		sessionID = "temp-" + go_pkg_utils.UUID()
		if err := configBot.Save(sessionID, "", "", false); err != nil {
			c.write(Frame{Type: FrameDone, UUID: f.UUID, Error: err.Error()})
			return
		}
	}
	if f.UUID != "" {
		c.own(f.UUID, sessionID)
	}

	execCtx := agentTypes.WithOrigin(context.WithoutCancel(ctx), "cli-")
	execCtx = agentTypes.WithWindowHash(execCtx, f.Rayload.WindowHash)
	execCtx = fast.With(execCtx, f.Rayload.Fast)
	execCtx = guide.With(execCtx, f.Rayload.Guide)
	content := strings.TrimSpace(f.Rayload.Input)
	data := exec.Prepare(exec.ExecuteMeta{
		Model:          f.Rayload.Model,
		Reasoning:      strings.TrimSpace(f.Rayload.Reasoning),
		WorkDir:        f.Rayload.WorkDir,
		Content:        content,
		Input:          content,
		SessionID:      sessionID,
		AllowAll:       f.Rayload.AllowAll,
		TUI:            true,
		PendingTask:    f.Rayload.PendingTask,
		HistoryContent: f.Rayload.HistoryContent,
	})

	events, wait := exec.Stream(execCtx, sessionID, 64, func(stream chan<- agentTypes.Event) error {
		return exec.Start(execCtx, data, stream)
	})
	for ev := range events {
		if ev.Type == agentTypes.EventTextDelta {
			continue
		}
		frame := Frame{Type: FrameEvent, UUID: f.UUID, SessionID: sessionID, Event: &ev}
		if ev.Err != nil {
			frame.Error = ev.Err.Error()
		}
		c.write(frame)
	}

	result := Frame{Type: FrameDone, UUID: f.UUID, SessionID: sessionID}
	if err := wait(); err != nil {
		result.Error = err.Error()
		result.Canceled = errors.Is(err, context.Canceled) || errors.Is(err, runtime.ErrUserCanceled)
	}
	c.write(result)
}
