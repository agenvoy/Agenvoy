package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

type Transcriber func(ctx context.Context, raw []byte, mime string) (string, error)

var (
	transcriberMu sync.RWMutex
	transcriber   Transcriber
)

var mediaMimeByExt = map[string]string{
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".mp3":  "audio/mp3",
	".wav":  "audio/wav",
	".m4a":  "audio/mp4",
	".flac": "audio/flac",
	".aac":  "audio/aac",
	".aiff": "audio/aiff",
	".mp4":  "video/mp4",
	".mov":  "video/mov",
	".webm": "video/webm",
	".mpg":  "video/mpeg",
	".mpeg": "video/mpeg",
	".3gp":  "video/3gpp",
}

func SetTranscriber(fn Transcriber) {
	transcriberMu.Lock()
	defer transcriberMu.Unlock()
	transcriber = fn
}

func IsMedia(path string) bool {
	_, ok := mediaMimeByExt[strings.ToLower(filepath.Ext(path))]
	return ok
}

func TranscribeMedia(ctx context.Context, path string) (string, error) {
	mime, ok := mediaMimeByExt[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", utilsmsg.New(utilsmsg.Invalid, "path", "unsupported file extension: %s", filepath.Ext(path))
	}

	transcriberMu.RLock()
	fn := transcriber
	transcriberMu.RUnlock()
	if fn == nil {
		return "", utilsmsg.New(utilsmsg.Unavailable, "transcriber", "not installed")
	}

	// * os.ReadFile retained: go-pkg/filesystem only exposes ReadText (string); audio/video need raw bytes.
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Internal, "os.ReadFile", "", err)
	}
	return fn(ctx, raw, mime)
}
