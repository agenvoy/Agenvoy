package tool

import (
	"path/filepath"
	"sort"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func Toggle(toolName, toolArgs string, workDir ...string) (bool, error) {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return false, utilsmsg.New(utilsmsg.Invalid, "tool name", "empty")
	}

	path := filesystem.AllowToolGlobalPath
	if len(workDir) > 0 {
		if strings.TrimSpace(workDir[0]) == "" {
			return false, utilsmsg.New(utilsmsg.Invalid, "work dir", "empty")
		}
		path = filesystem.AllowToolPath(workDir[0])
	}

	entry := toolName
	if subject := formatArgs(toolName, toolArgs); subject != "" {
		entry = toolName + "(" + formatGlob(subject) + ")"
	}

	allowMap := read(path)
	added := false
	if allowMap[entry] {
		delete(allowMap, entry)
	} else {
		allowMap[entry] = true
		added = true
	}
	return added, write(path, allowMap)
}

func ToggleMCP(prefix string, entries []string) error {
	if strings.TrimSpace(prefix) == "" {
		return utilsmsg.New(utilsmsg.Invalid, "prefix", "empty")
	}

	set := Load()
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

	return write(filesystem.AllowToolGlobalPath, set)
}

func write(path string, allowMap map[string]bool) error {
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.CheckDir", filepath.Dir(path), err)
	}

	lines := make([]string, 0, len(allowMap))
	for entry := range allowMap {
		lines = append(lines, entry)
	}

	sort.Strings(lines)
	var sb strings.Builder
	for _, entry := range lines {
		sb.WriteString(entry)
		sb.WriteByte('\n')
	}
	if err := go_pkg_filesystem.WriteFile(path, sb.String(), 0644); err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "go_pkg_filesystem.WriteFile", path, err)
	}
	return nil
}

func formatGlob(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) + 4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '*', '?', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
