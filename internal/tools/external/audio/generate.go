package audio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type Request struct {
	Text       string
	Voice      string
	OutputFile string
}

func Generate(ctx context.Context, req Request) (string, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return "", utilsmsg.New(utilsmsg.Invalid, "text", "required")
	}

	result, agentName, err := Speak(ctx, text, go_llm_router.TTSOptions{Voice: strings.TrimSpace(req.Voice), Format: speechFormat})
	if err != nil {
		return "", err
	}
	if len(result.Audio) == 0 {
		return "", utilsmsg.New(utilsmsg.Upstream, agentName, "returned no audio data")
	}

	path := outputPath(req.OutputFile, result.MimeType)
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.CheckDir", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, result.Audio, 0644); err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.WriteFile", path, err)
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
