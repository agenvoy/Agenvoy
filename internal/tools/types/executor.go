package toolTypes

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	apiAdapter "github.com/pardnchiu/agenvoy/internal/toolAdapter/api"
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
	Allowed          []string // * limit to these folders to use
	AllowedCommand   map[string]bool
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
}
