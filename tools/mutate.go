package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewMutateTool(exec sandbox.Executor, limiter *Limiter, maxCodeSize int, maxOutputBytes int64, contentMode ContentMode, metadata ToolMetadata) mcpserver.ServerTool {
	return newServerTool(ToolMutate, metadata.MutateTitle, metadata.MutateDescription, sandbox.ModeMutate, exec, limiter, maxCodeSize, maxOutputBytes, false, true, false, contentMode, metadata.AuthErrorHint)
}
