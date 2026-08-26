package tui

import (
	"strings"
	"unicode"

	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func quoteDroppedPaths(raw string) (string, bool) {
	if strings.ContainsAny(raw, `"'`) {
		return "", false
	}

	list := splitDroppedPaths(raw)
	if len(list) == 0 {
		return "", false
	}
	for _, path := range list {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~/") {
			return "", false
		}
		if !go_pkg_filesystem_reader.Exists(go_pkg_utils.AbsPath("", path)) {
			return "", false
		}
	}

	quoted := make([]string, 0, len(list))
	for _, path := range list {
		quoted = append(quoted, `"`+path+`"`)
	}
	return strings.Join(quoted, " "), true
}

func splitDroppedPaths(raw string) []string {
	var list []string
	var sb strings.Builder
	escaped := false
	for _, r := range raw {
		switch {
		case escaped:
			sb.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case unicode.IsSpace(r):
			if sb.Len() > 0 {
				list = append(list, sb.String())
				sb.Reset()
			}
		default:
			sb.WriteRune(r)
		}
	}
	if sb.Len() > 0 {
		list = append(list, sb.String())
	}
	return list
}
