package subagent

import (
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func sendEvent(parent chan<- agentTypes.Event, name string, ev agentTypes.Event) {
	if parent == nil {
		return
	}
	switch ev.Type {
	case agentTypes.EventDone,
		agentTypes.EventText,
		agentTypes.EventTextDone,
		agentTypes.EventAgentSelect,
		agentTypes.EventAgentResult,
		agentTypes.EventSummaryGenerate,
		agentTypes.EventToolCallStart,
		agentTypes.EventToolCallEnd,
		agentTypes.EventToolCallText,
		agentTypes.EventSkillResult,
		agentTypes.EventTodoUpdate:
		return
	}

	out := ev
	if out.Source == "" {
		out.Source = name
	}
	parent <- out
}
