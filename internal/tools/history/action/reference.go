package actionHistory

import "strings"

var referenceTool = map[string]bool{
	"fetch_page":     true,
	"search_web":     true,
	"subagents":      true,
	"http_request":   true,
	"generate_image": true,
	"download_file":  true,
}

var writeTool = map[string]bool{
	"edit_file":  true,
	"edit_skill": true,
	"edit_tool":  true,
}

func IsRetained(name string) bool {
	return isReference(name) || writeTool[name]
}

func isReference(name string) bool {
	if referenceTool[name] {
		return true
	}
	return strings.HasPrefix(name, "mcp__") ||
		strings.HasPrefix(name, "api_") ||
		strings.HasPrefix(name, "script_") ||
		strings.HasPrefix(name, "ext_")
}
