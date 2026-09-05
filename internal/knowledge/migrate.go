package knowledge

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
)

func Migrate() {
	migrateTorii()
	migrateFiles()
}

// * ensure v0.35.2: toriidb to sqlite
func migrateTorii() {
	db := torii.DB(torii.DBKnowledge)
	entries := db.Scan(context.Background(), "*", torii.ScanOption{})

	imported := 0
	for _, entry := range entries {
		if _, exists := Read(entry.Key); !exists {
			var record Record
			if err := json.Unmarshal([]byte(entry.Value()), &record); err != nil {
				slog.Debug("knowledge migrate: json.Unmarshal",
					slog.String("name", entry.Key),
					slog.String("error", err.Error()))
				continue
			}
			if err := Write(entry.Key, record.Content); err != nil {
				slog.Warn("knowledge migrate: Write",
					slog.String("name", entry.Key),
					slog.String("error", err.Error()))
				continue
			}
			imported++
		}
		db.Del(context.Background(), entry.Key)
	}

	if imported > 0 {
		slog.Info("knowledge migrated into SQLite",
			slog.Int("count", imported),
			slog.String("from", "toriidb"))
	}
}

// * ensure v0.32.4: file to toriidb
func migrateFiles() {
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
		slog.Info("knowledge migrated into SQLite",
			slog.Int("count", imported),
			slog.String("from", dir))
	}
}
