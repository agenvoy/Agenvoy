package knowledge

import (
	"log/slog"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

// * ensure v0.32.4 data can be migrated
func Migrate() {
	dir := filesystem.KnowledgeDir
	if !go_pkg_filesystem_reader.IsDir(dir) {
		return
	}

	files, err := go_pkg_filesystem_reader.ListFiles(dir)
	if err != nil {
		slog.Warn("knowledge migrate: ListFiles", slog.String("error", err.Error()))
		return
	}

	imported := 0
	for _, one := range files {
		name, ok := strings.CutSuffix(one.Name, ".md")
		if !ok || strings.HasPrefix(one.Name, ".") {
			continue
		}
		if _, exists := Read(name); exists {
			continue
		}

		path := filepath.Join(dir, one.Name)
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			slog.Warn("knowledge migrate: ReadText",
				slog.String("path", path),
				slog.String("error", err.Error()))
			continue
		}

		if err := Write(name, content); err != nil {
			slog.Warn("knowledge migrate: Write",
				slog.String("name", name),
				slog.String("error", err.Error()))
			continue
		}
		imported++
	}

	if imported > 0 {
		slog.Info("knowledge migrated into ToriiDB",
			slog.Int("count", imported),
			slog.String("from", dir))
	}
}
