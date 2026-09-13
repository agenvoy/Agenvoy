package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/interactive"
)

type pendingTaskInfo struct {
	TaskHash     string `json:"task_hash"`
	Objective    string `json:"objective"`
	HasQuestions bool   `json:"has_questions"`
}

func ListSessionPending() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		hashes := interactive.ListResumablePending(sid)
		list := make([]pendingTaskInfo, 0, len(hashes))
		for _, h := range hashes {
			info, ok := interactive.LoadPendingInfo(sid, h)
			if !ok {
				continue
			}
			list = append(list, pendingTaskInfo{
				TaskHash:     info.TaskHash,
				Objective:    info.Objective,
				HasQuestions: info.HasQuestions,
			})
		}
		c.JSON(http.StatusOK, gin.H{"pending": list})
	}
}

func DeleteSessionPending() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		taskHash := strings.TrimSpace(c.Param("task_hash"))
		if sid == "" || taskHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_hash are required"})
			return
		}

		if _, ok := interactive.LoadPendingInfo(sid, taskHash); !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "pending task not found"})
			return
		}

		interactive.DeletePending(sid, taskHash)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

type pendingQuestionInfo struct {
	Question    string   `json:"question"`
	Detail      string   `json:"detail,omitempty"`
	Options     []string `json:"options,omitempty"`
	MultiSelect bool     `json:"multi_select,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
}

func GetSessionPendingQuestions() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		taskHash := strings.TrimSpace(c.Param("task_hash"))
		if sid == "" || taskHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_hash are required"})
			return
		}

		questions, err := interactive.LoadPendingQuestions(sid, taskHash)
		if err != nil {
			c.JSON(http.StatusGone, gin.H{"error": "pending task already resolved in another session"})
			return
		}

		list := make([]pendingQuestionInfo, 0, len(questions))
		for _, q := range questions {
			list = append(list, pendingQuestionInfo{
				Question:    q.Question,
				Detail:      q.Detail,
				Options:     q.Options,
				MultiSelect: q.MultiSelect,
				Secret:      q.Secret,
			})
		}
		c.JSON(http.StatusOK, gin.H{"questions": list})
	}
}

func ResumeSessionPending() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		taskHash := strings.TrimSpace(c.Param("task_hash"))
		if sid == "" || taskHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and task_hash are required"})
			return
		}

		var body struct {
			Answers []any `json:"answers"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		info, ok := interactive.LoadPendingInfo(sid, taskHash)
		if !ok {
			c.JSON(http.StatusGone, gin.H{"error": "pending task already resolved in another session"})
			return
		}

		var content, historyContent string
		if info.HasQuestions && len(body.Answers) > 0 {
			full, history, err := interactive.LoadResumeMessage(sid, taskHash, body.Answers)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("load resume: %v", err)})
				return
			}
			content = full
			historyContent = history
		} else {
			full, history, err := interactive.LoadResumeMessage(sid, taskHash, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("load resume: %v", err)})
				return
			}
			content = full
			historyContent = history
		}

		ctx := context.WithoutCancel(c.Request.Context())
		events, _ := exec.Stream(ctx, sid, 64, func(stream chan<- agentTypes.Event) error {
			workDir, _ := os.UserHomeDir()
			trimContent := strings.TrimSpace(content)
			data := exec.Prepare(exec.ExecuteMeta{
				WorkDir:        workDir,
				Content:        trimContent,
				Input:          trimContent,
				SessionID:      sid,
				AllowAll:       interactive.LoadPendingAllowAll(sid, taskHash),
				PendingTask:    taskHash,
				HistoryContent: historyContent,
			})
			if err := exec.Start(agentTypes.WithOrigin(ctx, "chat-"), data, stream); err != nil {
				stream <- agentTypes.ErrorEvent(err)
			}
			return nil
		})

		result := collectResult(content, events)
		drainEvents(events)
		if result.Canceled {
			c.JSON(http.StatusOK, gin.H{
				"session_id": sid,
				"canceled":   true,
			})
			return
		}
		if result.Error != "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      result.Error,
				"session_id": sid,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"session_id": sid,
			"text":       result.Text,
		})
	}
}

type resumeResult struct {
	Text     string
	Error    string
	Canceled bool
}

func collectResult(_ string, events <-chan agentTypes.Event) resumeResult {
	var text strings.Builder
	var lastErr string
	timeout := time.After(time.Duration(filesystem.MaxResumeWaitMin) * time.Minute)

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return resumeResult{Text: text.String(), Error: lastErr}
			}
			switch ev.Type {
			case agentTypes.EventCanceled:
				return resumeResult{Text: text.String(), Canceled: true}
			case agentTypes.EventText, agentTypes.EventTextDone:
				if ev.Text != "" {
					text.WriteString(ev.Text)
				}
			case agentTypes.EventError, agentTypes.EventExecError:
				lastErr = ev.Text
				if ev.Err != nil {
					lastErr = ev.Err.Error()
				}
			}
		case <-timeout:
			return resumeResult{Error: "execution timeout"}
		}
	}
}
