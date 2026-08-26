package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	goRuntime "runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/tools/file/boundary"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const runCommandTimeout = 60 * time.Minute

func registRunCommand() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "run_command",
		Timeout:     runCommandTimeout,
		SystemUse:   false,
		AlwaysLoad:  true,
		AlwaysAllow: false,
		Concurrent:  false,
		Description: fmt.Sprintf(`Runs a binary in the work directory and returns its combined stdout/stderr.
Use for 跑一下 / 執行 / build / test / git, and for bash / shell / terminal.
Reading a file → read_files; finding one → find_files; %s; opening one in an app → open_file.`, systemPackageRoute()),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"argv": map[string]any{
					"type":        "array",
					"description": "The command as an argv array — ['git','status'], ['python3','script.py','--name','value with spaces']. Pipes, redirects and globbing need ['sh','-c','<full command>']; a plain command with no shell metacharacter (| && > * ~) is called directly, never wrapped in sh -c. ['cd','<path>'] switches the work directory for later calls, and the path is verified first. When the request names a capability rather than an exact command, resolve which binary is installed before running one: a single ['sh','-c','command -v <every candidate>'] prints only those that exist. Guessing the most common name costs one round trip per guess and the failure reads as 'command not found', not as 'wrong binary'.",
					"items":       map[string]any{"type": "string"},
					"minItems":    1,
				},
				"write_paths": map[string]any{
					"type":        "array",
					"description": "Absolute paths outside $HOME this command has to write to — ['/opt/homebrew'] for brew upgrade, ['/usr/local'] for a system install. The sandbox only allows writes under $HOME, so a command touching anything else fails with a permission error that looks like a file ownership problem and is not one. Each path needs the user's approval on this call. Paths under $HOME need not be listed.",
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

func deniedCommandErr(binary string) error {
	return fmt.Errorf("%s is on denied_command and can never run; it cannot be approved and retrying changes nothing", binary)
}

func systemPackageRoute() string {
	if goRuntime.GOOS == "linux" {
		return "installing or removing a system package → pkg_manage"
	}
	return `installing a system package → brew install, declaring write_paths: ["/opt/homebrew"]`
}

func runCommand(ctx context.Context, e *toolTypes.Executor, argv, writePaths []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("run_command requires a non-empty 'argv' array, e.g. [\"git\", \"status\"]")
	}

	binary := filepath.Base(argv[0])
	denied := filesystem.DeniedCommand

	if binary != argv[0] {
		return "", fmt.Errorf("failed to run command: %q must be a bare command name (%q), not a path", argv[0], binary)
	}

	if (binary == "sh" || binary == "bash") && len(argv) >= 3 && argv[1] == "-c" {
		if slices.Contains(denied, binary) {
			return "", deniedCommandErr(binary)
		}
		if strings.TrimSpace(argv[2]) == "" {
			return "", fmt.Errorf("%s -c requires a non-empty command string", binary)
		}
		if err := validateShellScript(argv[2], denied); err != nil {
			return "", err
		}
	} else {
		if binary == "cd" {
			return changeWorkDir(e, argv[1:])
		}
		if slices.Contains(denied, binary) {
			return "", deniedCommandErr(binary)
		}
		if binary == "rm" {
			return moveToTrash(ctx, e, argv[1:])
		}
	}

	ctx, cancel := context.WithTimeout(ctx, runCommandTimeout)
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

	sink := &progressWriter{send: toolTypes.Progress(ctx)}
	cmd.Stdout = sink
	cmd.Stderr = sink

	err = cmd.Run()
	output := sink.text()
	if err != nil {
		return fmt.Sprintf("%s\nError: %s%s", output, err.Error(), sandboxWriteHint(output, binds)), nil
	}

	return output, nil
}

const progressInterval = 500 * time.Millisecond

type progressWriter struct {
	mu      sync.Mutex
	buf     strings.Builder
	pending strings.Builder
	last    time.Time
	send    func(string)
}

func (w *progressWriter) Write(raw []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(raw)
	w.pending.Write(raw)

	if time.Since(w.last) < progressInterval {
		return len(raw), nil
	}
	line := lastLine(w.pending.String())
	w.pending.Reset()
	w.last = time.Now()
	if line != "" {
		w.send(line)
	}
	return len(raw), nil
}

func (w *progressWriter) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func lastLine(chunk string) string {
	for line := range strings.SplitSeq(strings.TrimRight(chunk, "\r\n"), "\n") {
		chunk = line
	}
	return strings.TrimSpace(go_pkg_utils.TruncateString(chunk, 200))
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
		if isSystemBinary(one) {
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

var systemBinDirs = []string{"/usr/bin/", "/bin/", "/usr/sbin/", "/sbin/", "/usr/libexec/"}

func isSystemBinary(path string) bool {
	return slices.ContainsFunc(systemBinDirs, func(d string) bool { return strings.HasPrefix(path, d) })
}

func stableRoot(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return path
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return path
	}
	return "/" + parts[0] + "/" + parts[1]
}

func shortestRoots(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, one := range list {
		root := stableRoot(one)
		if seen[root] {
			continue
		}
		seen[root] = true
		out = append(out, root)
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}
