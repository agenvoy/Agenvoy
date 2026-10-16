package skill

import (
	"maps"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func Match(workDir, skillName string) bool {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return false
	}
	return Load(workDir)[skillName]
}

func Load(workDir ...string) map[string]bool {
	allowMap := read(filesystem.AllowSkillGlobalPath)
	if len(workDir) > 0 && strings.TrimSpace(workDir[0]) != "" {
		maps.Copy(allowMap, read(filesystem.AllowSkillProjectPath(workDir[0])))
	}
	return allowMap
}

func read(path string) map[string]bool {
	allowMap := make(map[string]bool)
	if !go_pkg_filesystem_reader.Exists(path) {
		return allowMap
	}

	text, err := go_pkg_filesystem.ReadText(path)
	if err != nil {
		return allowMap
	}

	for line := range strings.SplitSeq(text, "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		allowMap[entry] = true
	}
	return allowMap
}
