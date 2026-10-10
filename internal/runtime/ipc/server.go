package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func Listen(ctx context.Context) (func(), error) {
	path := filesystem.DaemonSocketPath
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("os.Remove: %w", err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("os.Chmod: %w", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serve(ctx, conn)
		}
	}()

	return func() {
		listener.Close()
		os.Remove(path)
	}, nil
}

func serve(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	for {
		var f Frame
		if err := dec.Decode(&f); err != nil {
			return
		}
		if f.Type == FrameRun {
			run(ctx, enc, f)
		}
	}
}

func run(ctx context.Context, enc *json.Encoder, f Frame) {
	if f.Rayload == nil {
		enc.Encode(Frame{Type: FrameDone, Error: "run payload is required"})
		return
	}

	execCtx := agentTypes.WithOrigin(context.WithoutCancel(ctx), "cli-")
	content := strings.TrimSpace(f.Rayload.Input)
	data := exec.Prepare(exec.ExecuteMeta{
		Model:          f.Rayload.Model,
		Reasoning:      strings.TrimSpace(f.Rayload.Reasoning),
		WorkDir:        f.Rayload.WorkDir,
		Content:        content,
		Input:          content,
		SessionID:      f.SessionID,
		AllowAll:       f.Rayload.AllowAll,
		TUI:            true,
		PendingTask:    f.Rayload.PendingTask,
		HistoryContent: f.Rayload.HistoryContent,
	})

	events, wait := exec.Stream(execCtx, f.SessionID, 64, func(stream chan<- agentTypes.Event) error {
		return exec.Start(execCtx, data, stream)
	})
	for ev := range events {
		if ev.Type == agentTypes.EventTextDelta {
			continue
		}
		frame := Frame{Type: FrameEvent, Event: &ev}
		if ev.Err != nil {
			frame.Error = ev.Err.Error()
		}
		enc.Encode(frame)
	}

	result := Frame{Type: FrameDone}
	if err := wait(); err != nil {
		result.Error = err.Error()
	}
	enc.Encode(result)
}
