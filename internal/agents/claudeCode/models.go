package claudeCode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
)

const modelsRequestID = "agenvoy-models"

type initializeEvent struct {
	Type     string `json:"type"`
	Response struct {
		Subtype   string `json:"subtype"`
		RequestID string `json:"request_id"`
		Error     string `json:"error"`
		Response  struct {
			Models []struct {
				ResolvedModel string `json:"resolvedModel"`
			} `json:"models"`
		} `json:"response"`
	} `json:"response"`
}

func Models(ctx context.Context, _ provider.Config) ([]string, error) {
	if err := CheckBinary(); err != nil {
		return nil, err
	}

	request := fmt.Sprintf(`{"type":"control_request","request_id":%q,"request":{"subtype":"initialize"}}`+"\n", modelsRequestID)
	cmd := exec.CommandContext(ctx, "claude", "-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--safe-mode", "--tools", "", "--strict-mcp-config", "--no-session-persistence")
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(request)
	stderr := &limitedBuffer{}
	cmd.Stderr = stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude initialize: %w: %s", err, stderr.String())
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	for {
		var event initializeEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("claude initialize: no control_response in output")
			}
			return nil, fmt.Errorf("json.Decode: %w", err)
		}
		if event.Type != "control_response" || event.Response.RequestID != modelsRequestID {
			continue
		}
		if event.Response.Subtype != "success" {
			return nil, fmt.Errorf("claude initialize: %s", event.Response.Error)
		}

		ids := make([]string, 0, len(event.Response.Response.Models))
		for _, model := range event.Response.Response.Models {
			if model.ResolvedModel == "" || slices.Contains(ids, model.ResolvedModel) {
				continue
			}
			ids = append(ids, model.ResolvedModel)
		}
		return ids, nil
	}
}
