package ipc

import (
	"context"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

func (c *conn) mcp(ctx context.Context, f Frame) {
	result := Frame{Type: FrameMCP, UUID: f.UUID, MCP: &MCP{}}
	m := mcp.Manager()
	if f.MCP == nil || m == nil {
		result.Error = "mcp manager unavailable"
		c.write(result)
		return
	}

	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var err error
	switch f.MCP.Action {
	case "status":
		result.MCP.Servers = m.Status("")
	case "reconnect":
		err = m.ReconnectServer(opCtx, f.MCP.Server)
	case "tools":
		result.MCP.Tools, err = m.Tools(opCtx, f.MCP.Server)
	case "disconnect":
		m.Disconnect(f.MCP.Server)
	default:
		err = utilsmsg.New(utilsmsg.Invalid, "mcp action", "unknown %q", f.MCP.Action)
	}
	if err != nil {
		result.Error = err.Error()
	}
	c.write(result)
}
