package exec

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/session/summary"
	provider "github.com/pardnchiu/go-llm-router/core"
)

func GetSession(ctx context.Context, execData ExecuteMeta) (*agentTypes.AgentSession, error) {
	// * step1: reload skill list
	scanner := execData.SkillScanner
	if scanner == nil {
		scanner = agents.Scanner()
	}

	// * step2: assemble session history
	sessionID := strings.TrimSpace(execData.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("SessionID is required")
	}

	sessionDir := filesystem.SessionDir(sessionID)
	if !go_pkg_filesystem_reader.IsDir(sessionDir) {
		return nil, fmt.Errorf("session %q does not exist", sessionID)
	}

	oldHistory, maxHistory := sessionHistory.Get(sessionID)
	session := agentTypes.AgentSession{
		SystemPrompts: buildSystemPrompts(execData.WorkDir, execData.ExtraSystemPrompt, scanner, sessionID, execData.AllowAll, execData.ExcludeSkills, execData.ModelName()),
		Tools:         []provider.Message{},
		Histories:     sessionHistory.Messages(oldHistory),
		BaseLen:       len(oldHistory),
		OldHistories:  sessionHistory.Messages(maxHistory),
		ToolHistories: []provider.Message{},
	}
	if summary := summary.GetPrompt(sessionID, OldestMessageTime(maxHistory)); summary != "" {
		// * if summary not empty, add it
		session.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	// * step3: assemble user input
	userInput := strings.TrimSpace(execData.Input)
	if userInput == "" {
		userInput = strings.TrimSpace(execData.Content)
	}

	historyInput := userInput
	if content := strings.TrimSpace(execData.HistoryContent); content != "" {
		// * not save full resume input in history
		historyInput = content
	}

	session.Sender = execData.Sender
	session.UserSendAt = time.Now().UnixNano()
	// * add timestamp prefix to user input, for send at tracking
	prefix := sessionHistory.Record{
		SendAt: session.UserSendAt,
		Sender: session.Sender,
	}.Prefix()

	session.Histories = append(session.Histories, provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, historyInput),
	})
	session.UserInput = provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, buildInput(userInput, execData.ImageInputs, execData.FileInputs)),
	}
	SaveUserInputHistory(ctx, sessionID, historyInput)

	session.ID = sessionID
	return &session, nil
}

func OldestMessageTime(histories []sessionHistory.Record) time.Time {
	for _, record := range histories {
		if record.SendAt > 0 {
			return time.Unix(0, record.SendAt)
		}
	}
	return time.Time{}
}

func buildInput(userInput string, imageInputs []string, fileInputs []string) any {
	// * no file/image, directily return inpur
	if len(imageInputs) == 0 && len(fileInputs) == 0 {
		return userInput
	}

	parts := []provider.ContentPart{
		{Type: "text", Text: userInput},
	}

	// * if file is exist, append file content to user input
	for _, path := range fileInputs {
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			continue
		}
		parts = append(parts, provider.ContentPart{
			Type: "text",
			Text: fmt.Sprintf("---\npath: %s\n---\n%s", filepath.Base(path), content),
		})
	}

	// * if image is exist, append image content to user input
	for _, path := range imageInputs {
		b64, err := convertToBase64(path)
		if err != nil {
			continue
		}
		data := "data:image/jpeg;base64," + b64
		parts = append(parts, provider.ContentPart{
			Type:     "image_url",
			ImageURL: &provider.ImageURL{URL: data, Detail: "auto"},
		})
	}
	return parts
}

func convertToBase64(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("image.Decode: %w", err)
	}

	// * need to be use jpeg before send in claude/gemini model
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("jpeg.Encode: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
