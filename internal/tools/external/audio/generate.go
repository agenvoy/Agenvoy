package audio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

type Request struct {
	Text       string
	Voice      string
	OutputFile string
}

func Generate(ctx context.Context, req Request) (string, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return "", fmt.Errorf("text is required")
	}

	result, agentName, err := Speak(ctx, text, core.TTSOptions{Voice: strings.TrimSpace(req.Voice)})
	if err != nil {
		return "", err
	}
	if len(result.Audio) == 0 {
		return "", fmt.Errorf("%s returned no audio data", agentName)
	}

	path := outputPath(req.OutputFile)
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: CheckDir: %w", err)
	}
	if err := os.WriteFile(path, result.Audio, 0644); err != nil {
		return "", fmt.Errorf("os.WriteFile [%s]: %w", path, err)
	}

	return fmt.Sprintf("saved: %s\nmodel: %s\nbytes: %d", path, agentName, len(result.Audio)), nil
}

func outputPath(outputFile string) string {
	stem := filepath.Base(strings.TrimSpace(outputFile))
	stem = strings.Trim(strings.TrimSuffix(stem, filepath.Ext(stem)), `./\ `)
	if stem == "" {
		stem = "audio-" + time.Now().Format("20060102-150405")
	}
	return filepath.Join(filesystem.DownloadDir, stem+".wav")
}
