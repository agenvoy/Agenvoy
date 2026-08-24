package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func registRunCommand() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "run_command",
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: false,
		Concurrent:  false,
		Description: `Runs a binary in the work directory and returns its combined stdout/stderr.
Use for 跑一下 / 執行 / build / test / git, and for bash / shell / terminal.
Reading a file → read_files; finding one → find_files; installing a system binary → install_dependence; opening one in an app → open_file.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"argv": map[string]any{
					"type":        "array",
					"description": "The command as an argv array — ['git','status'], ['python3','script.py','--name','value with spaces']. Pipes, redirects and globbing need ['sh','-c','<full command>']; a plain command with no shell metacharacter (| && > * ~) is called directly, never wrapped in sh -c. ['cd','<path>'] switches the work directory for later calls, and the path is verified first.",
					"items":       map[string]any{"type": "string"},
					"minItems":    1,
				},
				"write_paths": map[string]any{
					"type":        "array",
					"description": "Absolute paths outside $HOME this command has to write to — ['/opt/homebrew'] for brew upgrade, ['/usr/local'] for a system install. The sandbox only allows writes under $HOME, so a command touching anything else fails with a permission error that looks like a file ownership problem and is not one. Each path needs the user's approval on this call, or an entry in path_white_list. Paths under $HOME need not be listed.",
					"items":       map[string]any{"type": "string"},
				},
			},
			"required": []string{"argv"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Argv       []string `json:"argv"`
				WritePaths []string `json:"write_paths"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			return runCommand(ctx, e, params.Argv, params.WritePaths)
		},
	})
}

const deniedHint = "run_command can never reach this path, with or without approval — retrying any shell command that names it fails the same way. find_files and read_files can reach it after the user approves a prompt; use those."

func runCommand(ctx context.Context, e *toolTypes.Executor, argv, writePaths []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("run_command requires a non-empty 'argv' array, e.g. [\"git\", \"status\"]")
	}

	joined := strings.Join(argv, " ")

	for _, dir := range filesystem.DeniedMap.Dirs {
		if strings.Contains(joined, "/"+dir+"/") || strings.Contains(joined, "/"+dir) || strings.Contains(joined, dir+"/") {
			return "", fmt.Errorf("access denied: %s. %s", dir, deniedHint)
		}
	}
	for _, f := range filesystem.DeniedMap.Files {
		if strings.Contains(joined, f) {
			return "", fmt.Errorf("access denied: %s. %s", f, deniedHint)
		}
	}

	binary := filepath.Base(argv[0])
	allowed := allowedWithGrants(e.SessionID, e.AllowedCommand)

	if binary != argv[0] {
		return "", fmt.Errorf("failed to run command: %q must be a bare command name (%q), not a path", argv[0], binary)
	}

	if (binary == "sh" || binary == "bash") && len(argv) >= 3 && argv[1] == "-c" {
		if !allowed[binary] {
			return "", fmt.Errorf("failed to run command: %s is not allowed", binary)
		}
		if strings.TrimSpace(argv[2]) == "" {
			return "", fmt.Errorf("%s -c requires a non-empty command string", binary)
		}
		if err := validateShellScript(argv[2], allowed); err != nil {
			return "", err
		}
	} else {
		if binary == "cd" {
			return changeWorkDir(e, argv[1:])
		}
		if !allowed[binary] {
			return "", fmt.Errorf("failed to run command: %s is not allowed", binary)
		}
		if binary == "rm" {
			return moveToTrash(ctx, e, argv[1:])
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	binds, err := boundary.WriteBinds(e.SessionID, e.WorkDir, writePaths)
	if err != nil {
		return "", err
	}
	var sandboxOpt *go_pkg_sandbox.Option
	if len(binds) > 0 {
		sandboxOpt = &go_pkg_sandbox.Option{
			MinimalBinds: &go_pkg_sandbox.BindSpec{
				WriteScope: go_pkg_sandbox.WriteHome,
				ReadWrite:  binds,
			},
		}
	}

	cmd, err := go_pkg_sandbox.Wrap(ctx, argv[0], argv[1:], e.WorkDir, sandboxOpt)
	if err != nil {
		return "", fmt.Errorf("sandbox.Wrap: %w", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\nError: %s%s", string(output), err.Error(), sandboxWriteHint(string(output), binds)), nil
	}

	return string(output), nil
}

const sandboxWriteAdvice = `

[agenvoy] the sandbox only allows writes under %s, which is why these read as unwritable: %s. They are writable outside the sandbox, so ownership is not the problem — do not run chown or chmod and do not ask the user to. Re-run this exact command with write_paths: [%s] and the user gets one prompt to approve those paths.`

var permissionMarkers = []string{
	"not writable", "permission denied", "operation not permitted",
	"read-only file system", "eacces",
}

var absPathPattern = regexp.MustCompile(`(^|[\s"'(])(/[A-Za-z0-9._@+\-/]{2,})`)

func sandboxWriteHint(output string, bound []string) string {
	lower := strings.ToLower(output)
	if !slices.ContainsFunc(permissionMarkers, func(m string) bool { return strings.Contains(lower, m) }) {
		return ""
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	seen := map[string]bool{}
	var list []string
	for _, match := range absPathPattern.FindAllStringSubmatch(output, -1) {
		one := strings.TrimRight(match[2], ".,:;)")
		if one == "" || seen[one] || strings.HasPrefix(one, home) {
			continue
		}
		if slices.ContainsFunc(bound, func(b string) bool { return one == b || strings.HasPrefix(one, b+"/") }) {
			continue
		}
		seen[one] = true
		list = append(list, one)
	}

	list = shortestRoots(list)
	if len(list) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(list))
	for _, one := range list {
		quoted = append(quoted, strconv.Quote(one))
	}
	return fmt.Sprintf(sandboxWriteAdvice, home, strings.Join(list, ", "), strings.Join(quoted, ", "))
}

func shortestRoots(list []string) []string {
	for range 4 {
		reduced := collapseOnce(list)
		if len(reduced) == len(list) {
			break
		}
		list = reduced
	}
	if len(list) > 5 {
		list = list[:5]
	}
	return list
}

func collapseOnce(list []string) []string {
	slices.SortFunc(list, func(a, b string) int { return len(a) - len(b) })

	siblings := map[string]int{}
	for _, one := range list {
		parent := filepath.Dir(one)
		if strings.Count(parent, "/") >= 2 {
			siblings[parent]++
		}
	}

	seen := map[string]bool{}
	var out []string
	for _, one := range list {
		if parent := filepath.Dir(one); siblings[parent] > 1 {
			one = parent
		}
		if seen[one] || slices.ContainsFunc(out, func(kept string) bool { return strings.HasPrefix(one, kept+"/") }) {
			continue
		}
		seen[one] = true
		out = append(out, one)
	}
	return out
}
