package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/config"
)

type contextKey string

const (
	sessionIDKey   contextKey = "session_id"
	executionIDKey contextKey = "execution_id"
	recorderKey    contextKey = "api_call_recorder"
)

type Logger struct {
	slog  *slog.Logger
	debug bool
}

type ExecutionRecord struct {
	Tool            string
	Code            string
	ChangeSummary   string
	Duration        time.Duration
	APICallCount    int
	ResultSizeBytes int
	Success         bool
	ErrorType       string
	ErrorMessage    string
	APICalls        []APICallRecord
	StackTrace      string
}

type APICallRecord struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status,omitempty"`
	ResponseSize int    `json:"responseSizeBytes,omitempty"`
	DurationMS   int64  `json:"durationMs"`
}

func NewLogger(w io.Writer, level config.LogLevel) *Logger {
	slogLevel := slog.LevelInfo
	if level == config.LogLevelDebug {
		slogLevel = slog.LevelDebug
	}
	return &Logger{
		slog: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slogLevel,
		})),
		debug: level == config.LogLevelDebug,
	}
}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

func WithExecutionID(ctx context.Context, executionID string) context.Context {
	return context.WithValue(ctx, executionIDKey, executionID)
}

type APICallRecorder struct {
	mu    sync.Mutex
	calls []APICallRecord
}

func WithAPICallRecorder(ctx context.Context) (context.Context, *APICallRecorder) {
	recorder := &APICallRecorder{}
	return context.WithValue(ctx, recorderKey, recorder), recorder
}

func RecordAPICall(ctx context.Context, record APICallRecord) {
	recorder, _ := ctx.Value(recorderKey).(*APICallRecorder)
	if recorder == nil {
		return
	}
	recorder.mu.Lock()
	recorder.calls = append(recorder.calls, record)
	recorder.mu.Unlock()
}

func (r *APICallRecorder) Snapshot() []APICallRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]APICallRecord, len(r.calls))
	copy(out, r.calls)
	return out
}

func (l *Logger) DebugEnabled() bool {
	return l != nil && l.debug
}

func (l *Logger) LogExecution(ctx context.Context, record ExecutionRecord) {
	if l == nil {
		return
	}

	status := "success"
	if !record.Success {
		status = "error"
	}

	attrs := []any{
		"session_id", sessionIDFromContext(ctx),
		"execution_id", executionIDFromContext(ctx),
		"tool", record.Tool,
		"change_summary", record.ChangeSummary,
		"code_hash", hashCode(record.Code),
		"duration_ms", record.Duration.Milliseconds(),
		"api_call_count", record.APICallCount,
		"result_size_bytes", record.ResultSizeBytes,
		"status", status,
	}
	if record.ErrorType != "" {
		attrs = append(attrs, "error_type", record.ErrorType)
	}
	if record.ErrorMessage != "" {
		attrs = append(attrs, "error_message", record.ErrorMessage)
	}
	if l.debug {
		attrs = append(attrs, "code", record.Code)
		if len(record.APICalls) > 0 {
			attrs = append(attrs, "api_calls", record.APICalls)
		}
		if record.StackTrace != "" {
			attrs = append(attrs, "stack_trace", record.StackTrace)
		}
	}

	l.slog.InfoContext(ctx, "tool execution", attrs...)
}

func CaptureStackTrace() string {
	return string(debug.Stack())
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func sessionIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(sessionIDKey).(string)
	return value
}

func executionIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(executionIDKey).(string)
	return value
}
