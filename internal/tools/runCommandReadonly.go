package tools

import (
	"context"
	"encoding/json"
	"fmt"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const runCommandReadonlyName = "run_command_readonly"

func registRunCommandReadonly() {
	toolRegister.Regist(toolRegister.Def{
		Name:        runCommandReadonlyName,
		Timeout:     runCommandTimeout,
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `Fills the gaps the file tools leave for read-only inspection: runs a command that only looks and returns its combined stdout/stderr — git status / log / diff / show / blame, du / stat / wc, which / command -v, ps, a CLI's version or config dump. It must not create, change, move or delete anything: no files, git state, packages, processes, settings or remote services. Runs without asking the user and in parallel with other calls.
Reading a file (cat / head / tail) → read_files; listing a directory, globbing or grepping content (ls / find / grep / rg) → find_files — those come first, this is not a shell substitute for them.
Anything that writes, installs, builds, tests, formats, commits, pushes or changes directory → run_command.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"argv": map[string]any{
					"type":        "array",
					"description": "The command as an argv array — ['git','log','--oneline','-5'], ['du','-sh','node_modules']. Pipes need ['sh','-c','<full command>'] and every command in the pipeline must be read-only too; a redirect that writes (> >>) disqualifies the call. A plain command with no shell metacharacter is called directly, never wrapped in sh -c.",
					"items":       map[string]any{"type": "string"},
					"minItems":    1,
				},
			},
			"required": []string{"argv"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Argv []string `json:"argv"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Argv) > 0 && params.Argv[0] == "cd" {
				return "", fmt.Errorf("cd switches the work directory shared by every call in this run, so it cannot go through %s, which runs in parallel; use run_command", runCommandReadonlyName)
			}
			return runCommand(ctx, e, params.Argv, nil)
		},
	})
}
