package agents

import (
	"sync"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

type RefreshFunc func() (agentTypes.Agent, agentTypes.Agent, agentTypes.ModelSet)

var (
	mu         sync.RWMutex
	dispatcher agentTypes.Agent
	summary    agentTypes.Agent
	registry   agentTypes.ModelSet
	scanner    *runtime.SkillScanner
	refresher  RefreshFunc

	loadMu sync.Mutex
	loaded bool
)

func Set(dispatcherBot agentTypes.Agent, summaryBot agentTypes.Agent, agentRegistry agentTypes.ModelSet, skillScanner *runtime.SkillScanner) {
	mu.Lock()
	defer mu.Unlock()

	dispatcher = dispatcherBot
	summary = summaryBot
	registry = agentRegistry
	scanner = skillScanner
}

func MarkLoaded() {
	loadMu.Lock()
	defer loadMu.Unlock()
	loaded = true
}

func SetRefresher(fn RefreshFunc) {
	mu.Lock()
	defer mu.Unlock()

	refresher = fn
}

func Reload() bool {
	mu.RLock()
	fn := refresher
	mu.RUnlock()
	if fn == nil {
		return false
	}

	dispatcherBot, summaryBot, agentRegistry := fn()
	mu.Lock()
	dispatcher = dispatcherBot
	summary = summaryBot
	registry = agentRegistry
	mu.Unlock()

	loadMu.Lock()
	loaded = true
	loadMu.Unlock()
	return true
}

func ensureLoaded() {
	loadMu.Lock()
	defer loadMu.Unlock()
	if loaded {
		return
	}

	mu.RLock()
	fn := refresher
	mu.RUnlock()
	if fn == nil {
		return
	}

	dispatcherBot, summaryBot, agentRegistry := fn()
	mu.Lock()
	dispatcher = dispatcherBot
	summary = summaryBot
	registry = agentRegistry
	mu.Unlock()
	loaded = true
}

func DispatcherBot() agentTypes.Agent {
	ensureLoaded()

	mu.RLock()
	defer mu.RUnlock()
	return dispatcher
}

func SummaryBot() agentTypes.Agent {
	ensureLoaded()

	mu.RLock()
	defer mu.RUnlock()
	return summary
}

func ModelSet() agentTypes.ModelSet {
	ensureLoaded()

	mu.RLock()
	defer mu.RUnlock()
	return registry
}

func Scanner() *runtime.SkillScanner {
	mu.RLock()
	defer mu.RUnlock()
	return scanner
}
