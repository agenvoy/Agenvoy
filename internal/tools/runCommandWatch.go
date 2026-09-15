package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	"mvdan.cc/sh/v3/syntax"
)

const maxWatchScriptDepth = 5

var watchBinaries = map[string]bool{
	"chokidar": true,
}

var packageRunners = map[string]bool{
	"npm": true, "pnpm": true, "yarn": true, "bun": true,
}

func watchCommandErr(command string) error {
	return fmt.Errorf("%q starts a file watcher that never exits; run_command waits for the process to end, so it would hang until the %s timeout. Run a one-shot build instead (e.g. the same tool without --watch, or the package script that builds once) and do not retry this command", command, runCommandTimeout)
}

func isWatchFlag(arg string) bool {
	return arg == "--watch" || strings.HasPrefix(arg, "--watch=")
}

func checkWatchArgs(args []string, dir string, depth int) error {
	if len(args) == 0 || depth > maxWatchScriptDepth {
		return nil
	}

	base := filepath.Base(args[0])
	if watchBinaries[base] || slices.ContainsFunc(args[1:], isWatchFlag) {
		return watchCommandErr(strings.Join(args, " "))
	}
	if (base == "sh" || base == "bash") && len(args) >= 3 && args[1] == "-c" {
		return checkWatchScript(args[2], dir, depth+1)
	}
	if !packageRunners[base] {
		return nil
	}

	name := packageScriptName(args[1:])
	if name == "" {
		return nil
	}
	script, pkgDir := findPackageScript(dir, name)
	if script == "" {
		return nil
	}
	if err := checkWatchScript(script, pkgDir, depth+1); err != nil {
		return fmt.Errorf("%q runs package script %q (%s): %w", strings.Join(args, " "), name, script, err)
	}
	return nil
}

func checkWatchScript(script, dir string, depth int) error {
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return nil
	}

	var bad error
	syntax.Walk(file, func(node syntax.Node) bool {
		if bad != nil {
			return false
		}
		call, ok := node.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		if _, ok := staticWord(call.Args[0]); !ok {
			return true
		}

		args := make([]string, 0, len(call.Args))
		for _, w := range call.Args {
			if s, ok := staticWord(w); ok {
				args = append(args, s)
			}
		}
		if filepath.Base(args[0]) == "cd" {
			if len(args) > 1 {
				dir = resolveWatchDir(dir, args[1])
			}
			return true
		}
		bad = checkWatchArgs(args, dir, depth)
		return bad == nil
	})
	return bad
}

func packageScriptName(args []string) string {
	var positional []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 {
		return ""
	}
	if positional[0] == "run" || positional[0] == "run-script" {
		if len(positional) < 2 {
			return ""
		}
		return positional[1]
	}
	return positional[0]
}

func findPackageScript(dir, name string) (string, string) {
	for {
		path := filepath.Join(dir, "package.json")
		if go_pkg_filesystem_reader.Exists(path) {
			pkg, err := go_pkg_filesystem.ReadJSON[struct {
				Scripts map[string]string `json:"scripts"`
			}](path)
			if err != nil {
				return "", ""
			}
			return pkg.Scripts[name], dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
}

func resolveWatchDir(dir, target string) string {
	if rest, ok := strings.CutPrefix(target, "~"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return dir
		}
		return filepath.Join(home, rest)
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Join(dir, target)
}
