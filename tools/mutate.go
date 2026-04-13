package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewMutateTool(exec sandbox.Executor, limiter *Limiter, maxOutputBytes int64, contentMode ContentMode) mcpserver.ServerTool {
	return newServerTool(ToolMutate, mutateTitle, mutateDescription, sandbox.ModeMutate, exec, limiter, maxOutputBytes, false, true, false, contentMode)
}
