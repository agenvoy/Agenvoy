package toolRegister

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	provider "github.com/pardnchiu/go-llm-router/core"
)

type Handler func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error)

type GroupHandler func(ctx context.Context, e *toolTypes.Executor, name string, args json.RawMessage) (string, error)

const DefaultToolTimeout = time.Minute

type Def struct {
	Name          string
	Description   string
	Parameters    map[string]any
	Handler       Handler
	AlwaysAllow   bool
	AlwaysLoad    bool
	Concurrent    bool
	FireAndForget bool
	Timeout       time.Duration
}

var handlerMap = map[string]Handler{}
var groupHandlerMap = map[string]GroupHandler{}
var defList []provider.Tool
var builtinNames []string
var readOnlySet = map[string]bool{}
var alwaysLoadSet = map[string]bool{}
var concurrentSet = map[string]bool{}
var fireAndForgetSet = map[string]bool{}
var timeoutMap = map[string]time.Duration{}

func Regist(d Def) {
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" {
		slog.Warn("toolRegister.Regist: empty name, skipped")
		return
	}

	if _, exists := handlerMap[d.Name]; exists {
		slog.Warn("toolRegister.Regist: name already registered, overwriting",
			slog.String("name", d.Name))
	}

	raw, _ := json.Marshal(d.Parameters)
	tool := provider.Tool{
		Type: "function",
		Function: provider.ToolFunction{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  raw,
		},
	}
	handlerMap[d.Name] = d.Handler
	defList = append(defList, tool)
	builtinNames = append(builtinNames, d.Name)
	if d.AlwaysAllow {
		readOnlySet[d.Name] = true
	}
	if d.AlwaysLoad {
		alwaysLoadSet[d.Name] = true
	}
	if d.Concurrent {
		concurrentSet[d.Name] = true
	}
	if d.FireAndForget {
		fireAndForgetSet[d.Name] = true
	}
	if d.Timeout > 0 {
		timeoutMap[d.Name] = d.Timeout
	}
}

func RemoveByPrefix(prefix string) {
	defList = slices.DeleteFunc(defList, func(t provider.Tool) bool {
		return strings.HasPrefix(t.Function.Name, prefix)
	})
	builtinNames = slices.DeleteFunc(builtinNames, func(n string) bool {
		return strings.HasPrefix(n, prefix)
	})
	for name := range handlerMap {
		if strings.HasPrefix(name, prefix) {
			delete(handlerMap, name)
			delete(readOnlySet, name)
			delete(alwaysLoadSet, name)
			delete(concurrentSet, name)
			delete(fireAndForgetSet, name)
			delete(timeoutMap, name)
		}
	}
}

func GetTimeout(name string) time.Duration {
	if t, ok := timeoutMap[name]; ok {
		return t
	}
	return DefaultToolTimeout
}

func IsAlwaysLoad(name string) bool {
	return alwaysLoadSet[name]
}

func IsReadOnly(name string) bool {
	return readOnlySet[name]
}

func MarkAlwaysAllow(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	readOnlySet[name] = true
}

func MarkConcurrent(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	concurrentSet[name] = true
}

func MarkTimeout(name string, timeout time.Duration) {
	name = strings.TrimSpace(name)
	if name == "" || timeout <= 0 {
		return
	}
	timeoutMap[name] = timeout
}

func IsConcurrent(name string) bool {
	return concurrentSet[name]
}

func IsFireAndForget(name string) bool {
	return fireAndForgetSet[name]
}

func GetTool(name string) *provider.Tool {
	for i := range defList {
		if defList[i].Function.Name == name {
			return &defList[i]
		}
	}
	return nil
}

func BuiltinNames() []string {
	dst := make([]string, len(builtinNames))
	copy(dst, builtinNames)
	return dst
}

func JSON() []byte {
	raw, err := json.Marshal(defList)
	if err != nil {
		return []byte("[]")
	}
	return raw
}

func Dispatch(ctx context.Context, e *toolTypes.Executor, name string, args json.RawMessage) (string, error) {
	timeout := GetTimeout(name)
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var run func() (string, error)
	if handler, ok := handlerMap[name]; ok {
		run = func() (string, error) { return handler(tctx, e, args) }
	} else {
		for prefix, groupHandler := range groupHandlerMap {
			if strings.HasPrefix(name, prefix) {
				handler := groupHandler
				run = func() (string, error) { return handler(tctx, e, name, args) }
				break
			}
		}
	}
	if run == nil {
		return "", fmt.Errorf("not exist: %s", name)
	}

	return runWithDeadline(tctx, run, name, timeout)
}

type dispatchResult struct {
	result string
	err    error
}

func runWithDeadline(tctx context.Context, run func() (string, error), name string, timeout time.Duration) (string, error) {
	done := make(chan dispatchResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("tool handler panicked",
					slog.String("name", name),
					slog.Any("panic", r))
				done <- dispatchResult{err: fmt.Errorf("tool %q panicked: %v", name, r)}
			}
		}()
		result, err := run()
		done <- dispatchResult{result: result, err: err}
	}()

	select {
	case out := <-done:
		if tctx.Err() == context.DeadlineExceeded {
			return out.result, fmt.Errorf("tool %q timed out after %s", name, timeout)
		}
		return out.result, out.err
	case <-tctx.Done():
		if errors.Is(tctx.Err(), context.DeadlineExceeded) {
			slog.Warn("tool handler abandoned after timeout",
				slog.String("name", name),
				slog.String("timeout", timeout.String()))
			return "", fmt.Errorf("tool %q timed out after %s and did not honor cancellation", name, timeout)
		}
		return "", tctx.Err()
	}
}

func RegistGroup(prefix string, handler GroupHandler) {
	groupHandlerMap[prefix] = handler
}
