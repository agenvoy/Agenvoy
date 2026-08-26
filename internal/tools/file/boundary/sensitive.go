package boundary

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func IsDeniedPath(absPath string) bool {
	absPath = filepath.Clean(strings.TrimSpace(absPath))
	if absPath == "" || absPath == "." {
		return false
	}
	sep := string(filepath.Separator)
	return slices.ContainsFunc(filesystem.DeniedPath, func(one string) bool {
		return absPath == one || strings.HasPrefix(absPath, one+sep)
	})
}

func IsSensitivePath(absPath string) bool {
	absPath = filepath.Clean(strings.TrimSpace(absPath))
	if absPath == "" || absPath == "." {
		return false
	}
	dic := filesystem.SensitivePath

	for segment := range strings.SplitSeq(absPath, string(filepath.Separator)) {
		if slices.Contains(dic.Dirs, segment) {
			return true
		}
	}

	base := filepath.Base(absPath)
	if slices.Contains(dic.Files, base) {
		return true
	}
	if slices.ContainsFunc(dic.Prefixes, func(p string) bool { return strings.HasPrefix(base, p) }) {
		return true
	}
	return slices.Contains(dic.Extensions, strings.ToLower(filepath.Ext(base)))
}
