package tool

import (
	"encoding/json"
	"maps"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func Match(workDir, toolName, toolArgs string) bool {
	allowMap := Load(workDir)
	if len(allowMap) == 0 {
		return false
	}

	args := formatArgs(toolName, toolArgs)
	for e := range allowMap {
		tool, ok := parse(e)
		if !ok || !tool.name.MatchString(toolName) {
			continue
		}
		if !tool.hasArg || tool.argGlob.MatchString(args) {
			return true
		}
	}
	return false
}

func Load(workDir ...string) map[string]bool {
	allowMap := read(filesystem.AllowToolGlobalPath)
	if len(workDir) > 0 && strings.TrimSpace(workDir[0]) != "" {
		maps.Copy(allowMap, read(filesystem.AllowToolPath(workDir[0])))
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

func formatArgs(toolName, toolArgs string) string {
	toolArgs = strings.TrimSpace(toolArgs)
	if toolArgs == "" {
		return ""
	}
	switch toolName {
	case "run_command":
		var p struct {
			Argv []string `json:"argv"`
		}
		if err := json.Unmarshal([]byte(toolArgs), &p); err == nil && len(p.Argv) > 0 {
			return strings.Join(p.Argv, " ")
		}
	}
	var generic map[string]any
	if err := json.Unmarshal([]byte(toolArgs), &generic); err == nil {
		for _, key := range []string{"path", "url", "target"} {
			if v, ok := generic[key].(string); ok && v != "" {
				return v
			}
		}
	}
	return toolArgs
}
