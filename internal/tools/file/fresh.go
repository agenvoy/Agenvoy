package file

import (
	"fmt"
	"os"
	"time"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func requireFresh(e *toolTypes.Executor, absPath string, modTime time.Time) error {
	readAt, ok := e.ReadModTime(absPath)
	if !ok {
		return fmt.Errorf("%s has not been read in this turn; read_files it first and build the edit from its current bytes. Nothing was written", absPath)
	}
	if !modTime.Equal(readAt) {
		return fmt.Errorf("%s changed on disk after it was read (modified %s, read version %s), by the user, a formatter or another command; read_files it again before editing. Nothing was written",
			absPath, modTime.Format(time.RFC3339Nano), readAt.Format(time.RFC3339Nano))
	}
	return nil
}

func markWritten(e *toolTypes.Executor, absPath string) {
	if info, err := os.Stat(absPath); err == nil {
		e.MarkRead(absPath, info.ModTime())
	}
}
