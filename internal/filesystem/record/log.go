package record

import (
	"bytes"
	"fmt"
	"os"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func TrimLog() error {
	file, err := os.OpenFile(filesystem.DaemonLogPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile [%s]: %w", filesystem.DaemonLogPath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("file.Stat [%s]: %w", filesystem.DaemonLogPath, err)
	}
	if stat.Size() <= filesystem.DocumentMaxBytes {
		return nil
	}

	raw := make([]byte, filesystem.DocumentMaxBytes*3/4)
	if _, err := file.ReadAt(raw, stat.Size()-filesystem.DocumentMaxBytes*3/4); err != nil {
		return fmt.Errorf("file.ReadAt [%s]: %w", filesystem.DaemonLogPath, err)
	}
	if i := bytes.IndexByte(raw, '\n'); i >= 0 {
		raw = raw[i+1:]
	}

	if _, err := file.WriteAt(raw, 0); err != nil {
		return fmt.Errorf("file.WriteAt [%s]: %w", filesystem.DaemonLogPath, err)
	}
	if err := file.Truncate(int64(len(raw))); err != nil {
		return fmt.Errorf("file.Truncate [%s]: %w", filesystem.DaemonLogPath, err)
	}
	return nil
}
