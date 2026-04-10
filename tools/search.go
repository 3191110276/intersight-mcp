package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewSearchTool(exec sandbox.Executor) mcpserver.ServerTool {
	return newServerTool(ToolSearch, searchTitle, searchDescription, sandbox.ModeSearch, exec, nil, true, false)
}
