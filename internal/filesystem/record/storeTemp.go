package record

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const storeTempMaxAge = 28 * 24 * time.Hour

var stampedEntry = regexp.MustCompile(`_(\d{8}_\d{6}\.\d{3})(?:_\d+)?(?:\.[^.]*)?$`)

func CleanStoreTemp() {
	root := filesystem.StoreTempDir
	if root == "" || !go_pkg_filesystem_reader.IsDir(root) {
		return
	}

	expiredAt := time.Now().Add(-storeTempMaxAge)

	var expired []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}

		match := stampedEntry.FindAllStringSubmatch(entry.Name(), -1)
		if match == nil {
			return nil
		}

		stamp, parseErr := time.ParseInLocation(filesystem.TrashStampLayout, match[len(match)-1][1], time.Local)
		if parseErr != nil || stamp.After(expiredAt) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		expired = append(expired, path)
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		slog.Debug("filepath.WalkDir",
			slog.String("dir", root),
			slog.String("error", err.Error()))
	}

	for _, path := range expired {
		if err := os.RemoveAll(path); err != nil {
			slog.Debug("os.RemoveAll",
				slog.String("path", path),
				slog.String("error", err.Error()))
		}
	}
	pruneEmptyTempDirs(root)
}

func pruneEmptyTempDirs(root string) {
	var dirs []string
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || path == root || !entry.IsDir() {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	})

	slices.SortFunc(dirs, func(a, b string) int { return len(b) - len(a) })
	for _, dir := range dirs {
		empty, err := go_pkg_filesystem_reader.IsEmpty(dir)
		if err != nil || !empty {
			continue
		}
		if err := os.Remove(dir); err != nil {
			slog.Debug("os.Remove",
				slog.String("path", dir),
				slog.String("error", err.Error()))
		}
	}
}
