package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewQueryTool(exec sandbox.Executor, limiter *Limiter, maxCodeSize int, maxOutputBytes int64, exposeMetricsApps bool, contentMode ContentMode, metadata ToolMetadata) mcpserver.ServerTool {
	return newServerTool(ToolQuery, metadata.QueryTitle, metadata.QueryDescription, sandbox.ModeQuery, exec, limiter, maxCodeSize, maxOutputBytes, true, false, exposeMetricsApps, contentMode, metadata.AuthErrorHint)
}
