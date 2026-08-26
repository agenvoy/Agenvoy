package interactive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	goRuntime "runtime"
	"slices"
	"strings"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var packageName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9+._-]*$`)

type pkgAction struct {
	needsPackage bool
	needsRoot    bool
}

var pkgActions = map[string]pkgAction{
	"install": {needsPackage: true, needsRoot: true},
	"remove":  {needsPackage: true, needsRoot: true},
	"update":  {needsPackage: false, needsRoot: true},
	"upgrade": {needsPackage: false, needsRoot: true},
	"search":  {needsPackage: true, needsRoot: false},
	"info":    {needsPackage: true, needsRoot: false},
}

var pkgCommands = map[string]map[string][]string{
	"apt": {
		"install": {"install", "-y"},
		"remove":  {"remove", "-y"},
		"update":  {"update"},
		"upgrade": {"upgrade", "-y"},
		"search":  {"search"},
		"info":    {"show"},
	},
	"apt-get": {
		"install": {"install", "-y"},
		"remove":  {"remove", "-y"},
		"update":  {"update"},
		"upgrade": {"upgrade", "-y"},
		"search":  nil,
		"info":    nil,
	},
	"dnf": {
		"install": {"install", "-y"},
		"remove":  {"remove", "-y"},
		"update":  {"makecache"},
		"upgrade": {"upgrade", "-y"},
		"search":  {"search"},
		"info":    {"info"},
	},
	"yum": {
		"install": {"install", "-y"},
		"remove":  {"remove", "-y"},
		"update":  {"makecache"},
		"upgrade": {"upgrade", "-y"},
		"search":  {"search"},
		"info":    {"info"},
	},
	"pacman": {
		"install": {"-S", "--noconfirm"},
		"remove":  {"-R", "--noconfirm"},
		"update":  {"-Sy"},
		"upgrade": {"-Su", "--noconfirm"},
		"search":  {"-Ss"},
		"info":    {"-Si"},
	},
	"apk": {
		"install": {"add"},
		"remove":  {"del"},
		"update":  {"update"},
		"upgrade": {"upgrade"},
		"search":  {"search"},
		"info":    {"info"},
	},
}

func RestrictedPkgManage(toolName, toolArgs string) []string {
	if toolName != "pkg_manage" {
		return nil
	}
	var p struct {
		Action  string `json:"action"`
		Package string `json:"package"`
	}
	if json.Unmarshal([]byte(toolArgs), &p) != nil {
		return nil
	}

	action := strings.TrimSpace(p.Action)
	if action == "" {
		action = "install"
	}
	if !pkgActions[action].needsRoot {
		return nil
	}

	label := "sudo: package manager " + action
	if pkg := strings.TrimSpace(p.Package); pkg != "" {
		label += " " + pkg
	}
	return []string{label}
}

func registPkgManage() {
	if goRuntime.GOOS != "linux" {
		return
	}

	toolRegister.Regist(toolRegister.Def{
		Name:        "pkg_manage",
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: false,
		Concurrent:  false,
		Description: `Drives the Linux package manager (apt / dnf / yum / pacman / apk) outside the sandbox, so the root operations bwrap cannot grant still work.
Use for 安裝 / 移除套件 / 更新套件庫 / command not found / 缺 ffmpeg 之類的執行檔.
run_command cannot do this: sudo is powerless inside bwrap. Language runtimes (node / python) → run_command with mise, fnm or uv; language-level packages (pip / npm / cargo) → run_command.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"install", "remove", "update", "upgrade", "search", "info"},
					"description": "install and remove act on one package; update refreshes the index; upgrade upgrades every installed package; search and info are read-only lookups. update and upgrade take no package.",
					"default":     "install",
				},
				"package": map[string]any{
					"type":        "string",
					"description": "The bare package name — 'ffmpeg', 'g++', 'ImageMagick'. No flags, no version pin, no second package. Required for every action except update and upgrade.",
				},
			},
			"required": []string{"action"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Action  string `json:"action"`
				Package string `json:"package"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json Unmarshal: %w", err)
			}

			action := strings.TrimSpace(params.Action)
			if action == "" {
				action = "install"
			}
			spec, ok := pkgActions[action]
			if !ok {
				return "", fmt.Errorf("unknown action %q; available: %s", action, strings.Join(slices.Sorted(maps.Keys(pkgActions)), ", "))
			}

			pkg := strings.TrimSpace(params.Package)
			switch {
			case spec.needsPackage && pkg == "":
				return "", fmt.Errorf("package is required when action=%s", action)
			case !spec.needsPackage && pkg != "":
				return "", fmt.Errorf("action=%s takes no package, got %q", action, pkg)
			case pkg != "" && !packageName.MatchString(pkg):
				return "", fmt.Errorf("package %q must match %s — a bare package name, no flags and no version pin", pkg, packageName)
			}

			if action == "install" {
				if path, err := exec.LookPath(pkg); err == nil {
					return jsonString(map[string]any{
						"ok":                true,
						"action":            action,
						"name":              pkg,
						"already_installed": true,
						"path":              path,
					})
				}
			}

			cmd, arg, via, needsSudo, err := buildCommand(action, pkg, spec)
			if err != nil {
				return "", err
			}

			if needsSudo {
				arg = append([]string{"-n"}, arg...)
			}
			raw, runErr := exec.CommandContext(ctx, cmd, arg...).CombinedOutput()
			stderrMsg := trimOutput(string(raw))
			exitCode := 0
			if runErr != nil {
				exitCode = -1
				var exitErr *exec.ExitError
				if errors.As(runErr, &exitErr) {
					exitCode = exitErr.ExitCode()
				} else {
					stderrMsg = strings.TrimSpace(stderrMsg + "\n" + runErr.Error())
				}
			}

			if exitCode != 0 {
				result := map[string]any{
					"ok":        false,
					"action":    action,
					"name":      pkg,
					"via":       via,
					"exit_code": exitCode,
				}
				if strings.Contains(stderrMsg, "a password is required") {
					result["message"] = "sudo has no valid ticket in this process even though the call was approved with a password; approve it again, or grant this user NOPASSWD for " + via
				}
				if stderrMsg != "" {
					result["stderr"] = stderrMsg
				} else if result["message"] == nil {
					result["message"] = fmt.Sprintf("%s command %q exited with code %d (no output captured)", action, cmd, exitCode)
				}
				return jsonString(result)
			}

			out := map[string]any{
				"ok":     true,
				"action": action,
				"via":    via,
			}
			if pkg != "" {
				out["name"] = pkg
			}
			if stderrMsg != "" {
				out["stderr"] = stderrMsg
			}
			if action != "install" {
				return jsonString(out)
			}

			path, err := exec.LookPath(pkg)
			if err != nil {
				out["ok"] = false
				out["message"] = "install command succeeded but binary still not found in PATH"
				return jsonString(out)
			}
			out["path"] = path
			return jsonString(out)
		},
	})
}

func jsonString(dic map[string]any) (string, error) {
	raw, err := json.Marshal(dic)
	if err != nil {
		return "", fmt.Errorf("json Marshal: %w", err)
	}
	return string(raw), nil
}

func buildCommand(action, name string, spec pkgAction) (string, []string, string, bool, error) {
	if goRuntime.GOOS != "linux" {
		return "", nil, "", false, fmt.Errorf("unsupported OS: %s (pkg_manage only serves the Linux package managers)", goRuntime.GOOS)
	}

	isRoot := os.Geteuid() == 0
	for _, binary := range []string{"apt", "apt-get", "dnf", "yum", "pacman", "apk"} {
		if _, err := exec.LookPath(binary); err != nil {
			continue
		}
		verb := pkgCommands[binary][action]
		if len(verb) == 0 {
			return "", nil, "", false, fmt.Errorf("%s has no %s subcommand on this system; use run_command for that lookup", binary, action)
		}

		argv := append([]string{binary}, verb...)
		if spec.needsPackage {
			argv = append(argv, name)
		}
		if !spec.needsRoot || isRoot {
			return argv[0], argv[1:], binary, false, nil
		}
		if _, err := exec.LookPath("sudo"); err != nil {
			return "", nil, "", false, fmt.Errorf("sudo not found and current user is not root; cannot elevate %s for %s", action, binary)
		}
		return "sudo", argv, binary, true, nil
	}
	return "", nil, "", false, fmt.Errorf("no supported package manager (apt/apt-get/dnf/yum/pacman/apk) found on this Linux system")
}
func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	const max = 4096
	if len(s) > max {
		return "...(truncated)\n" + s[len(s)-max:]
	}
	return s
}
