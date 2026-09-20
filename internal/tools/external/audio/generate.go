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

	result, agentName, err := Speak(ctx, text, core.TTSOptions{Voice: strings.TrimSpace(req.Voice), Format: speechFormat})
	if err != nil {
		return "", err
	}
	if len(result.Audio) == 0 {
		return "", fmt.Errorf("%s returned no audio data", agentName)
	}

	path := outputPath(req.OutputFile, result.MimeType)
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: CheckDir: %w", err)
	}
	if err := os.WriteFile(path, result.Audio, 0644); err != nil {
		return "", fmt.Errorf("os.WriteFile [%s]: %w", path, err)
	}

	return fmt.Sprintf("saved: %s\nmodel: %s\nbytes: %d", path, agentName, len(result.Audio)), nil
}

const speechFormat = "opus"

var audioExtByMime = map[string]string{
	"audio/ogg":   ".ogg",
	"audio/opus":  ".ogg",
	"audio/wav":   ".wav",
	"audio/x-wav": ".wav",
	"audio/mpeg":  ".mp3",
	"audio/mp3":   ".mp3",
	"audio/aac":   ".aac",
	"audio/flac":  ".flac",
}

func audioExt(mime string) string {
	base, _, _ := strings.Cut(mime, ";")
	if ext, ok := audioExtByMime[strings.ToLower(strings.TrimSpace(base))]; ok {
		return ext
	}
	return ".wav"
}

func outputPath(outputFile, mime string) string {
	stem := filepath.Base(strings.TrimSpace(outputFile))
	stem = strings.Trim(strings.TrimSuffix(stem, filepath.Ext(stem)), `./\ `)
	if stem == "" {
		stem = "audio-" + time.Now().Format("20060102-150405")
	}
	return filepath.Join(filesystem.OutputDir(), stem+audioExt(mime))
}
