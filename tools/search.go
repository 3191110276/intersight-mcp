package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewSearchTool(exec sandbox.Executor, limiter *Limiter, maxCodeSize int, maxOutputBytes int64, contentMode ContentMode, metadata ToolMetadata) mcpserver.ServerTool {
	return newServerTool(ToolSearch, metadata.SearchTitle, metadata.SearchDescription, sandbox.ModeSearch, exec, limiter, maxCodeSize, maxOutputBytes, true, false, false, contentMode, metadata.AuthErrorHint)
}
