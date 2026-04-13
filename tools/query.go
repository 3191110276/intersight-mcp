package tools

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func NewQueryTool(exec sandbox.Executor, limiter *Limiter, maxOutputBytes int64, exposeMetricsApps bool, contentMode ContentMode) mcpserver.ServerTool {
	return newServerTool(ToolQuery, queryTitle, queryDescription, sandbox.ModeQuery, exec, limiter, maxOutputBytes, true, false, exposeMetricsApps, contentMode)
}
