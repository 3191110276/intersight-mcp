package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/sandbox"
	"github.com/mimaurer/intersight-mcp/tools"
)

const (
	defaultServerName    = "intersight-mcp"
	defaultServerVersion = "0.0.0"
)

type RuntimeConfig struct {
	ServerName        string
	ServerVersion     string
	MaxConcurrent     int
	ExposeMetricsApps bool
	Logger            *internalpkg.Logger

	SearchExecutor sandbox.Executor
	QueryExecutor  sandbox.Executor
	MutateExecutor sandbox.Executor
}

type Runtime struct {
	server *mcpserver.MCPServer
	stdio  *mcpserver.StdioServer
	search toolsCloser
	query  toolsCloser
	mutate toolsCloser
}

type toolsCloser interface {
	Close() error
}

func NewRuntime(cfg RuntimeConfig) (*Runtime, error) {
	if cfg.SearchExecutor == nil {
		return nil, errors.New("search executor is required")
	}
	if cfg.QueryExecutor == nil {
		return nil, errors.New("query executor is required")
	}
	if cfg.MutateExecutor == nil {
		return nil, errors.New("mutate executor is required")
	}
	if cfg.ServerName == "" {
		cfg.ServerName = defaultServerName
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = defaultServerVersion
	}

	srv := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithRecovery(),
		mcpserver.WithResourceCapabilities(false, false),
	)
	serverTools := tools.ServerTools(cfg.SearchExecutor, cfg.QueryExecutor, cfg.MutateExecutor, tools.NewLimiter(cfg.MaxConcurrent), cfg.ExposeMetricsApps)
	for i := range serverTools {
		serverTools[i].Handler = wrapToolHandler(serverTools[i].Tool.Name, serverTools[i].Handler, cfg.Logger)
	}
	srv.AddTools(serverTools...)

	stdio := mcpserver.NewStdioServer(srv)
	stdio.SetErrorLogger(log.New(io.Discard, "", 0))

	return &Runtime{
		server: srv,
		stdio:  stdio,
		search: cfg.SearchExecutor,
		query:  cfg.QueryExecutor,
		mutate: cfg.MutateExecutor,
	}, nil
}

func (r *Runtime) MCPServer() *mcpserver.MCPServer {
	if r == nil {
		return nil
	}
	return r.server
}

func (r *Runtime) Listen(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	if r == nil {
		return errors.New("runtime is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := r.stdio.Listen(ctx, &cancelOnEOFReader{reader: stdin, cancel: cancel}, stdout)
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return nil
	}
	return err
}

func (r *Runtime) Close() error {
	var firstErr error
	for _, exec := range []toolsCloser{r.search, r.query, r.mutate} {
		if exec == nil {
			continue
		}
		if err := exec.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type cancelOnEOFReader struct {
	reader io.Reader
	cancel context.CancelFunc
}

var executionSeq atomic.Uint64

func (r *cancelOnEOFReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if errors.Is(err, io.EOF) && r.cancel != nil {
		r.cancel()
	}
	return n, err
}

func enrichExecutionContext(ctx context.Context) context.Context {
	if session := mcpserver.ClientSessionFromContext(ctx); session != nil {
		ctx = internalpkg.WithSessionID(ctx, session.SessionID())
	}
	executionID := fmt.Sprintf("exec-%d", executionSeq.Add(1))
	return internalpkg.WithExecutionID(ctx, executionID)
}

func wrapToolHandler(tool string, next mcpserver.ToolHandlerFunc, logger *internalpkg.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		ctx = enrichExecutionContext(ctx)
		ctx, recorder := internalpkg.WithAPICallRecorder(ctx)

		result, err := next(ctx, request)
		if logger != nil {
			record := internalpkg.ExecutionRecord{
				Tool:            tool,
				Code:            request.GetString("code", ""),
				ChangeSummary:   request.GetString("changeSummary", ""),
				Duration:        time.Since(start),
				ResultSizeBytes: resultSize(toolResultValue(result)),
				Success:         err == nil && !toolResultIsError(result),
				APICalls:        recorder.Snapshot(),
			}
			record.APICallCount = len(record.APICalls)
			if err != nil {
				record.ErrorType = errorType(err)
				record.ErrorMessage = err.Error()
				if logger.DebugEnabled() {
					record.StackTrace = internalpkg.CaptureStackTrace()
				}
			} else if result != nil && result.IsError {
				if envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope); ok {
					record.ErrorType = envelope.Error.Type
					record.ErrorMessage = envelope.Error.Message
				}
			}
			logger.LogExecution(ctx, record)
		}
		return result, err
	}
}

func resultSize(value any) int {
	if value == nil {
		return 0
	}
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(data)
}

func toolResultValue(result *mcp.CallToolResult) any {
	if result == nil || result.StructuredContent == nil {
		return nil
	}
	if envelope, ok := result.StructuredContent.(contracts.SuccessEnvelope); ok {
		return envelope.Result
	}
	if envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope); ok {
		return envelope
	}
	return result.StructuredContent
}

func toolResultIsError(result *mcp.CallToolResult) bool {
	return result != nil && result.IsError
}

func errorType(err error) string {
	switch {
	case err == nil:
		return ""
	default:
		var authErr contracts.AuthError
		if errors.As(err, &authErr) {
			return contracts.ErrorTypeAuth
		}
		var httpErr contracts.HTTPError
		if errors.As(err, &httpErr) {
			return contracts.ErrorTypeHTTP
		}
		var networkErr contracts.NetworkError
		if errors.As(err, &networkErr) {
			return contracts.ErrorTypeNetwork
		}
		var timeoutErr contracts.TimeoutError
		if errors.As(err, &timeoutErr) {
			return contracts.ErrorTypeTimeout
		}
		var limitErr contracts.LimitError
		if errors.As(err, &limitErr) {
			return contracts.ErrorTypeLimit
		}
		var validationErr contracts.ValidationError
		if errors.As(err, &validationErr) {
			return contracts.ErrorTypeValidation
		}
		var referenceErr contracts.ReferenceError
		if errors.As(err, &referenceErr) {
			return contracts.ErrorTypeReference
		}
		var outputErr contracts.OutputTooLarge
		if errors.As(err, &outputErr) {
			return contracts.ErrorTypeOutputTooBig
		}
		return contracts.ErrorTypeInternal
	}
}
