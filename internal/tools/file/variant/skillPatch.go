package variant

import (
	"context"
	"fmt"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func patchSkillFile(ctx context.Context, e *toolTypes.Executor, path, oldString, newString string, replaceAll bool) (string, error) {
	if oldString == "" {
		return "", fmt.Errorf("old_string is required when mode=patch")
	}
	if oldString == newString {
		return "", fmt.Errorf("no edit needed")
	}

	absPath, err := skillPath(path)
	if err != nil {
		return "", err
	}

	change := capture(absPath)
	if err := patch(absPath, oldString, newString, replaceAll); err != nil {
		return "", err
	}
	record(ctx, e, change, "", "edit_skill")

	return fmt.Sprintf("updated: %s", absPath), nil
}
