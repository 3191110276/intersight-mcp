package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewSearchTool(exec sandbox.Executor, limiter *Limiter, maxOutputBytes int64, contentMode ContentMode) mcpserver.ServerTool {
	return newServerTool(ToolSearch, searchTitle, searchDescription, sandbox.ModeSearch, exec, limiter, maxOutputBytes, true, false, false, contentMode)
}
