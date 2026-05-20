package utils

import (
	"encoding/json"
	"fmt"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func formatToolEvent(count int, event agentTypes.Event) string {
	body := event.ToolName + "(" + TruncateStatus(event.ToolArgs) + ")"
	switch event.ToolName {
	case "fetch_page":
		body = formatFetchPage(event.ToolArgs)
	}
	return fmt.Sprintf("[tool #%d] %s", count, body)
}

func formatFetchPage(args string) string {
	var p struct {
		Link string `json:"link"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "Fetch(" + TruncateStatus(args) + ")"
	}
	return "Fetch(" + p.Link + " " + p.Type + ")"
}
