package toolSearcher

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func isMCPExposed(name string) bool {
	switch {
	case strings.HasPrefix(name, "script_"),
		strings.HasPrefix(name, "api_"),
		strings.HasPrefix(name, "ext_"):
		return true
	}
	return slices.Contains(toolRegister.BuiltinNames(), name)
}

func listTools(e *toolTypes.Executor, mcpOnly, system bool) (string, error) {
	list := make([]Tool, 0, len(e.AllTools))
	for _, tool := range e.AllTools {
		name := tool.Function.Name
		if mcpOnly && !isMCPExposed(name) {
			continue
		}
		if !system && toolRegister.IsSystemUse(name) {
			continue
		}
		list = append(list, Tool{
			Name:          name,
			Description:   tool.Function.Description,
			SystemDefault: strings.HasPrefix(strings.TrimSpace(tool.Function.Description), systemDefaultMarker),
		})
	}

	raw, err := json.Marshal(list)
	if err != nil {
		return "", fmt.Errorf("encoding/json: Marshal: %w", err)
	}
	return string(raw), nil
}
