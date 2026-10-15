package skill

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func Toggle(skillName string, workDir ...string) (bool, error) {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return false, fmt.Errorf("[Invalid] skill name: empty")
	}

	path := filesystem.AllowSkillGlobalPath
	if len(workDir) > 0 {
		if strings.TrimSpace(workDir[0]) == "" {
			return false, fmt.Errorf("[Invalid] work dir: empty")
		}
		path = filesystem.AllowSkillProjectPath(workDir[0])
	}

	allowMap := read(path)
	added := false
	if allowMap[skillName] {
		delete(allowMap, skillName)
	} else {
		allowMap[skillName] = true
		added = true
	}
	return added, write(path, allowMap)
}

func write(path string, allowMap map[string]bool) error {
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: CheckDir: %w", err)
	}

	lines := make([]string, 0, len(allowMap))
	for name := range allowMap {
		lines = append(lines, name)
	}

	sort.Strings(lines)
	var sb strings.Builder
	for _, name := range lines {
		sb.WriteString(name)
		sb.WriteByte('\n')
	}
	if err := go_pkg_filesystem.WriteFile(path, sb.String(), 0644); err != nil {
		return fmt.Errorf("github.com/pardnchiu/go-pkg/filesystem: WriteFile: %w", err)
	}
	return nil
}
