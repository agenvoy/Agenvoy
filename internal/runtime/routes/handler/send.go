package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	"github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

type Request struct {
	Content      string   `json:"content"`
	SSE          bool     `json:"sse"`
	SessionID    string   `json:"session_id"`
	Model        string   `json:"model,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`
	Persist      bool     `json:"persist,omitempty"`
	Chat         bool     `json:"chat,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	WorkDir      string   `json:"work_dir,omitempty"`
	Skill        string   `json:"skill,omitempty"`
}

func Send() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
			return
		}

		workDir, err := resolveWorkDir(req.WorkDir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sessionID := req.SessionID
		if sessionID == "" {
			prefix := "temp-"
			switch {
			case req.Chat:
				prefix = "chat-"
			case req.Persist:
				prefix = "http-"
			}
			sessionID = prefix + utils.UUID()
		}

		if err := configBot.Save(sessionID, "", "", false); err != nil {
			slog.Debug("sessionBot Save",
				slog.String("session", sessionID),
				slog.String("error", err.Error()))
		}

		trimContent := strings.TrimSpace(req.Content)
		data := exec.Prepare(exec.ExecuteMeta{
			Model:             req.Model,
			WorkDir:           workDir,
			SkillName:         req.Skill,
			Content:           trimContent,
			Input:             trimContent,
			SessionID:         sessionID,
			ExcludeTools:      req.ExcludeTools,
			ExtraSystemPrompt: req.SystemPrompt,
		})

		if exec.IsRunning(sessionID) {
			exec.AppendSteer(sessionID, req.Content)
			c.JSON(http.StatusOK, gin.H{
				"session_id": sessionID,
				"steer":      true,
			})
			return
		}

		execCtx := agentTypes.WithOrigin(context.WithoutCancel(c.Request.Context()), "chat-")
		events, _ := exec.Stream(execCtx, sessionID, 64, func(stream chan<- agentTypes.Event) error {
			withFollowup(execCtx, sessionID, stream, func(wrapped chan<- agentTypes.Event) {
				if err := exec.Start(execCtx, data, wrapped); err != nil {
					wrapped <- agentTypes.ErrorEvent(err)
				}
			})
			return nil
		})

		if req.SSE {
			sendSSE(c, sessionID, req.Content, events)
		} else {
			sendResult(c, sessionID, req.Content, events)
		}
		drainEvents(events)
	}
}

func resolveWorkDir(input string) (string, error) {
	home, _ := os.UserHomeDir()
	dir := strings.TrimSpace(input)
	if dir == "" {
		return home, nil
	}

	resolved, err := go_pkg_filesystem.AbsPath(home, dir, go_pkg_filesystem.AbsPathOption{})
	if err != nil {
		return "", fmt.Errorf("work_dir %q cannot be resolved: %w", dir, err)
	}
	if !go_pkg_filesystem_reader.Exists(resolved) {
		return "", fmt.Errorf("work_dir %q does not exist", dir)
	}
	if !go_pkg_filesystem_reader.IsDir(resolved) {
		return "", fmt.Errorf("work_dir %q is not a directory", dir)
	}
	return resolved, nil
}
