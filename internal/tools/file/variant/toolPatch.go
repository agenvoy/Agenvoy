package variant

import (
	"context"
	"fmt"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func patchToolFile(ctx context.Context, e *toolTypes.Executor, name, tag, oldString, newString string, replaceAll bool) (string, error) {
	if oldString == "" {
		return "", fmt.Errorf("old_string is required when mode=patch")
	}
	if oldString == newString {
		return "", fmt.Errorf("no edit needed")
	}

	target, err := toolTarget(name, tag, true)
	if err != nil {
		return "", err
	}

	change := capture(target)
	if err := patch(target, oldString, newString, replaceAll); err != nil {
		return "", err
	}
	record(ctx, e, change, "", "edit_tool")

	return fmt.Sprintf("updated: %s", target), nil
}
