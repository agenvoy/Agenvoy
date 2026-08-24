package boundary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

type pathSet struct {
	mu    sync.Mutex
	paths map[string]struct{}
}

var cache sync.Map

func get(sessionID string) *pathSet {
	if v, ok := cache.Load(sessionID); ok {
		return v.(*pathSet)
	}
	ps := &pathSet{paths: map[string]struct{}{}}
	actual, _ := cache.LoadOrStore(sessionID, ps)
	return actual.(*pathSet)
}

func Grant(sessionID string, list ...string) {
	if len(list) == 0 {
		return
	}
	ps := get(sessionID)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, one := range list {
		one = filepath.Clean(strings.TrimSpace(one))
		if one == "" || one == "." || one == string(filepath.Separator) {
			continue
		}
		ps.paths[one] = struct{}{}
	}
}

func isGranted(sessionID, absPath string) bool {
	v, ok := cache.Load(sessionID)
	if !ok {
		return false
	}
	ps := v.(*pathSet)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	_, hit := ps.paths[filepath.Clean(absPath)]
	return hit
}

func resolveRaw(baseDir, path string) (string, error) {
	path = strings.TrimSpace(path)
	switch {
	case path == "":
		path = baseDir
	case path == "~" || strings.HasPrefix(path, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("os.UserHomeDir: %w", err)
		}
		path = filepath.Join(home, path[1:])
	case !filepath.IsAbs(path):
		path = filepath.Join(baseDir, path)
	}

	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("filepath.Abs: %w", err)
		}
		path = abs
	}
	return go_pkg_filesystem.RealPath(path)
}

func isWhitelisted(absPath string) bool {
	if go_pkg_filesystem.IsDenied(absPath) {
		return false
	}
	sep := string(filepath.Separator)
	for _, one := range filesystem.PathWhiteList {
		if absPath == one || strings.HasPrefix(absPath, one+sep) {
			return true
		}
	}
	return false
}

func Resolve(sessionID, baseDir, path string) (string, error) {
	abs, err := go_pkg_filesystem.AbsPath(baseDir, path, go_pkg_filesystem.AbsPathOption{HomeOnly: true})
	if err == nil {
		return abs, nil
	}

	restricted, rawErr := resolveRaw(baseDir, path)
	if rawErr != nil {
		return "", err
	}
	if !isGranted(sessionID, restricted) && !isWhitelisted(restricted) {
		return "", fmt.Errorf("%w; the user was not asked or did not approve this path, so retrying the same path fails the same way", err)
	}
	return restricted, nil
}

func Restricted(workDir, toolName, toolArgs string) []string {
	list := rawPaths(toolName, toolArgs)
	if len(list) == 0 {
		return nil
	}

	base := baseDir(toolName, workDir)
	seen := map[string]bool{}
	var out []string
	for _, one := range list {
		one = strings.TrimSpace(one)
		if one == "" {
			continue
		}
		if _, err := go_pkg_filesystem.AbsPath(base, one, go_pkg_filesystem.AbsPathOption{HomeOnly: true}); err == nil {
			continue
		}

		abs, err := resolveRaw(base, one)
		if err != nil || seen[abs] || isWhitelisted(abs) {
			continue
		}

		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

func baseDir(toolName, workDir string) string {
	if toolName == "fetch_page" || strings.TrimSpace(workDir) == "" {
		return filesystem.DownloadDir
	}
	return workDir
}

func rawPaths(toolName, toolArgs string) []string {
	switch toolName {
	case "read_files":
		var p struct {
			Files []struct {
				Path string `json:"path"`
			} `json:"files"`
		}
		if json.Unmarshal([]byte(toolArgs), &p) != nil {
			return nil
		}
		out := make([]string, 0, len(p.Files))
		for _, f := range p.Files {
			out = append(out, f.Path)
		}
		return out

	case "edit_file", "open_file":
		var p struct {
			Path string `json:"path"`
		}
		if json.Unmarshal([]byte(toolArgs), &p) != nil {
			return nil
		}
		return []string{p.Path}

	case "find_files":
		var p struct {
			Queries []struct {
				Dir string `json:"dir"`
			} `json:"queries"`
		}
		if json.Unmarshal([]byte(toolArgs), &p) != nil {
			return nil
		}
		out := make([]string, 0, len(p.Queries))
		for _, q := range p.Queries {
			out = append(out, q.Dir)
		}
		return out

	case "file_history":
		var p struct {
			Path  string   `json:"path"`
			Paths []string `json:"paths"`
		}
		if json.Unmarshal([]byte(toolArgs), &p) != nil {
			return nil
		}
		return append([]string{p.Path}, p.Paths...)

	case "fetch_page":
		var p struct {
			Save   bool   `json:"save"`
			SaveTo string `json:"save_to"`
		}
		if json.Unmarshal([]byte(toolArgs), &p) != nil || !p.Save {
			return nil
		}
		return []string{p.SaveTo}
	}
	return nil
}
