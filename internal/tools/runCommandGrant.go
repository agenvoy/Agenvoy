package tools

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/syntax"
)

type commandSet struct {
	mu   sync.Mutex
	bins map[string]bool
}

var commandGrants sync.Map

func GrantCommands(sessionID string, list []string) {
	if len(list) == 0 {
		return
	}
	v, ok := commandGrants.Load(sessionID)
	if !ok {
		v, _ = commandGrants.LoadOrStore(sessionID, &commandSet{bins: map[string]bool{}})
	}
	cs := v.(*commandSet)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for _, one := range list {
		one = strings.TrimSpace(one)
		if one == "" {
			continue
		}
		cs.bins[one] = true
	}
}

func allowedWithGrants(sessionID string, base map[string]bool) map[string]bool {
	v, ok := commandGrants.Load(sessionID)
	if !ok {
		return base
	}
	cs := v.(*commandSet)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if len(cs.bins) == 0 {
		return base
	}
	out := make(map[string]bool, len(base)+len(cs.bins))
	maps.Copy(out, base)
	maps.Copy(out, cs.bins)
	return out
}

func RestrictedCommands(allowed map[string]bool, toolName, toolArgs string) []string {
	if toolName != "run_command" {
		return nil
	}
	var p struct {
		Argv []string `json:"argv"`
	}
	if json.Unmarshal([]byte(toolArgs), &p) != nil || len(p.Argv) == 0 {
		return nil
	}

	argv := p.Argv
	binary := filepath.Base(argv[0])
	if binary == "cd" {
		return nil
	}
	if binary != argv[0] {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] || allowed[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}

	add(binary)
	if (binary == "sh" || binary == "bash") && len(argv) >= 3 && argv[1] == "-c" {
		for _, one := range scriptCommands(argv[2]) {
			add(one)
		}
	}
	return out
}

func scriptCommands(script string) []string {
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return nil
	}

	var out []string
	syntax.Walk(file, func(node syntax.Node) bool {
		call, ok := node.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		bin, ok := staticWord(call.Args[0])
		if !ok {
			return true
		}
		base := filepath.Base(bin)
		if base != bin || shellAllow[base] || base == "rm" {
			return true
		}
		out = append(out, base)

		if (base == "sh" || base == "bash") && len(call.Args) >= 3 {
			if flag, ok := staticWord(call.Args[1]); ok && flag == "-c" {
				if inner, ok := staticWord(call.Args[2]); ok {
					out = append(out, scriptCommands(inner)...)
				}
			}
		}
		return true
	})
	return out
}
