package allowTool

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func LoadGlobal() map[string]bool {
	out := make(map[string]bool)
	if !go_pkg_filesystem_reader.Exists(filesystem.AllowToolGlobalPath) {
		return out
	}
	text, err := go_pkg_filesystem.ReadText(filesystem.AllowToolGlobalPath)
	if err != nil {
		return out
	}
	for line := range strings.SplitSeq(text, "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		out[entry] = true
	}
	return out
}

func ReplaceGlobalPrefix(prefix string, entries []string) error {
	if strings.TrimSpace(prefix) == "" {
		return fmt.Errorf("empty prefix")
	}

	set := LoadGlobal()
	for entry := range set {
		if strings.HasPrefix(entry, prefix) {
			delete(set, entry)
		}
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			set[entry] = true
		}
	}

	names := make([]string, 0, len(set))
	for entry := range set {
		names = append(names, entry)
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, entry := range names {
		sb.WriteString(entry)
		sb.WriteByte('\n')
	}
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(filesystem.AllowToolGlobalPath), true); err != nil {
		return fmt.Errorf("CheckDir: %w", err)
	}
	if err := go_pkg_filesystem.WriteFile(filesystem.AllowToolGlobalPath, sb.String(), 0644); err != nil {
		return fmt.Errorf("WriteFile: %w", err)
	}
	return nil
}
