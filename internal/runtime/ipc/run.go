package ipc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/fast"
	"github.com/pardnchiu/agenvoy/internal/agents/exec/guide"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
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

	c.addWindow(f.Rayload.WindowHash)
	runningMu.Lock()
	runningConn[sessionID] = c
	runningMu.Unlock()
	defer func() {
		runningMu.Lock()
		if runningConn[sessionID] == c {
			delete(runningConn, sessionID)
		}
		runningMu.Unlock()
	}()
	execCtx := agentTypes.WithOrigin(ctx, "cli-")
	execCtx = exec.WithBotPushPrefix(execCtx, go_pkg_utils.TruncateString(f.Rayload.Input, 32))
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

	events, wait := exec.Stream(execCtx, sessionID, 64, func(stream chan<- agentTypes.Event) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("exec.Start panic: %v", r)
			}
		}()
		return exec.Start(execCtx, data, stream)
	})
	terminated := false
	taskHash := ""
	for ev := range events {
		if ev.Type == agentTypes.EventTextDelta {
			continue
		}
		switch ev.Type {
		case agentTypes.EventDone, agentTypes.EventCanceled, agentTypes.EventError:
			terminated = true
		}
		if ev.TaskHash != "" && ev.TaskHash != taskHash {
			taskHash = ev.TaskHash
			c.askMu.Lock()
			c.tasks[taskHash] = true
			c.askMu.Unlock()
			defer func(hash string) {
				c.askMu.Lock()
				delete(c.tasks, hash)
				c.askMu.Unlock()
			}(taskHash)
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
		if !terminated {
			ev := agentTypes.Event{Type: agentTypes.EventError, Text: err.Error(), TaskHash: taskHash, WindowHash: f.Rayload.WindowHash}
			if result.Canceled {
				ev.Type = agentTypes.EventCanceled
			}
			pubsub.Pub(sessionID, ev)
		}
	}
	c.write(result)
}
