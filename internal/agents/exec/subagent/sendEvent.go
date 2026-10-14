package subagent

import (
	agenttypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func sendEvent(parent chan<- agenttypes.Event, name string, ev agenttypes.Event) {
	if parent == nil {
		return
	}
	switch ev.Type {
	case agenttypes.EventDone,
		agenttypes.EventText,
		agenttypes.EventTextDone,
		agenttypes.EventAgentSelect,
		agenttypes.EventAgentResult,
		agenttypes.EventSummaryGenerate,
		agenttypes.EventToolCallStart,
		agenttypes.EventToolCallEnd,
		agenttypes.EventToolCallText,
		agenttypes.EventSkillResult,
		agenttypes.EventTodoUpdate:
		return
	}

	out := ev
	if out.Source == "" {
		out.Source = name
	}
	parent <- out
}
