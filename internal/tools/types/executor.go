package toolTypes

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/pardnchiu/agenvoy/internal/runtime"
	apiAdapter "github.com/pardnchiu/agenvoy/internal/runtime/toolAdapter/api"
	provider "github.com/pardnchiu/go-llm-router/core"
)

type ScriptToolExecutor interface {
	IsExist(name string) bool
	Execute(ctx context.Context, name string, args json.RawMessage, workDir string) (string, error)
	GetTools() []map[string]any
}

type Executor struct {
	ToolsMu          sync.Mutex
	WorkDir          string
	SessionID        string
	Tools            []provider.Tool
	AllTools         []provider.Tool
	StubTools        map[string]bool
	ExcludeTools     map[string]bool
	APIToolbox       *apiAdapter.Adapter
	ScriptToolbox    ScriptToolExecutor
	ExtAPIToolbox    *apiAdapter.Adapter
	ExtScriptToolbox ScriptToolExecutor

	SkillScanner    *runtime.SkillScanner
	CancelExecution context.CancelFunc
	PendingTask     string
	IgnoreHistory   bool
	filesMu         sync.Mutex
	filesEdited     []string
}

func (e *Executor) RecordFile(path string) {
	if e == nil || strings.TrimSpace(path) == "" {
		return
	}
	e.filesMu.Lock()
	defer e.filesMu.Unlock()
	if slices.Contains(e.filesEdited, path) {
		return
	}
	e.filesEdited = append(e.filesEdited, path)
}

func (e *Executor) SeedFiles(paths []string) {
	if e == nil {
		return
	}
	for _, one := range paths {
		e.RecordFile(one)
	}
}

func (e *Executor) EditedFiles() []string {
	if e == nil {
		return nil
	}
	e.filesMu.Lock()
	defer e.filesMu.Unlock()
	return slices.Clone(e.filesEdited)
}
