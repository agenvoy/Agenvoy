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

func buildContent(content string, imageInputs []string, fileInputs []string) any {
	if len(imageInputs) == 0 && len(fileInputs) == 0 {
		return content
	}

	parts := []provider.ContentPart{
		{
			Type: "text",
			Text: content,
		},
	}

	for _, path := range fileInputs {
		text, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			continue
		}
		parts = append(parts, provider.ContentPart{
			Type: "text",
			Text: fmt.Sprintf("---\npath: %s\n---\n%s", filepath.Base(path), text),
		})
	}

	for _, path := range imageInputs {
		b64, err := convertToBase64(path)
		if err != nil {
			continue
		}
		dataURL := "data:image/jpeg;base64," + b64
		parts = append(parts, provider.ContentPart{
			Type:     "image_url",
			ImageURL: &provider.ImageURL{URL: dataURL, Detail: "auto"},
		})
	}
	return parts
}

func GetSession(ctx context.Context, execData ExecuteMeta) (*agentTypes.AgentSession, error) {
	scanner := execData.SkillScanner
	if scanner == nil {
		scanner = agents.Scanner()
	}
	trimInput := strings.TrimSpace(execData.Content)
	session := agentTypes.AgentSession{
		Tools:     []provider.Message{},
		Histories: []provider.Message{},
	}

	overrideID := strings.TrimSpace(execData.SessionID)
	if overrideID == "" {
		return nil, fmt.Errorf("execData.SessionID is required")
	}
	sessionDir := filesystem.SessionDir(overrideID)
	if !go_pkg_filesystem_reader.IsDir(sessionDir) {
		return nil, fmt.Errorf("session %q does not exist", overrideID)
	}

	oldHistory, maxHistory := sessionHistory.Get(overrideID)
	session.Histories = sessionHistory.Messages(oldHistory)
	session.BaseLen = len(session.Histories)

	session.SystemPrompts = BuildSystemPrompts(execData.WorkDir, execData.ExtraSystemPrompt, scanner, overrideID, execData.AllowAll, execData.ExcludeSkills, execData.ModelName())
	if summary := summary.GetPrompt(overrideID, OldestMessageTime(maxHistory)); summary != "" {
		session.SummaryMessage = provider.Message{Role: "user", Content: summary}
	}

	session.OldHistories = sessionHistory.Messages(maxHistory)
	session.ToolHistories = []provider.Message{}

	userText := strings.TrimSpace(execData.Input)
	if userText == "" {
		userText = trimInput
	}
	histText := userText
	if h := strings.TrimSpace(execData.HistoryContent); h != "" {
		histText = h
	}

	session.Sender = execData.Sender
	session.UserSendAt = time.Now().UnixNano()
	prefix := sessionHistory.Record{
		SendAt: session.UserSendAt,
		Sender: session.Sender,
	}.Prefix()

	session.Histories = append(session.Histories, provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, histText),
	})
	session.UserInput = provider.Message{
		Role:    "user",
		Content: sessionHistory.WithPrefix(prefix, buildContent(userText, execData.ImageInputs, execData.FileInputs)),
	}
	SaveUserInputHistory(ctx, overrideID, histText)

	session.ID = overrideID
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

func convertToBase64(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
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
