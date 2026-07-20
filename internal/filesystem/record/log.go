package record

import (
	"fmt"
	"os"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
)

const (
	maxSize    = 1 << 20
	trimToSize = 768 << 10
)

func TrimLog() error {
	stat, err := os.Stat(filesystem.DaemonLogPath)
	if err != nil {
		return fmt.Errorf("os.Stat [%s]: %w", filesystem.DaemonLogPath, err)
	}
	if stat.Size() <= maxSize {
		return nil
	}

	content, err := go_pkg_filesystem.ReadText(filesystem.DaemonLogPath)
	if err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem.ReadText [%s]: %w", filesystem.DaemonLogPath, err)
	}

	raw := []byte(content)
	if int64(len(raw)) <= maxSize {
		return nil
	}

	result := max(len(raw)-trimToSize, 0)
	for result < len(raw) && raw[result] != '\n' {
		result++
	}
	if result < len(raw) {
		result++
	}

	if err := go_pkg_filesystem.WriteText(filesystem.DaemonLogPath, string(raw[result:])); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem.WriteText [%s]: %w", filesystem.DaemonLogPath, err)
	}
	return nil
}
