package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewSearchTool(exec sandbox.Executor, limiter *Limiter) mcpserver.ServerTool {
	return newServerTool(ToolSearch, searchTitle, searchDescription, sandbox.ModeSearch, exec, limiter, true, false, false)
}
