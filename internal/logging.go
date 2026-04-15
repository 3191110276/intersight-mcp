package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"log/slog"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/internal/config"
)

type contextKey string

const (
	sessionIDKey   contextKey = "session_id"
	executionIDKey contextKey = "execution_id"
	recorderKey    contextKey = "api_call_recorder"
)

type Logger struct {
	slog              *slog.Logger
	debug             bool
	includeUnsafeCode bool
	redactors         []codeRedactor
}

type LoggerOptions struct {
	Redactions []implementations.LogRedaction
}

type codeRedactor struct {
	pattern *regexp.Regexp
	replace string
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

func NewLogger(w io.Writer, level config.LogLevel, includeUnsafeCode bool, options LoggerOptions) *Logger {
	slogLevel := slog.LevelInfo
	if level == config.LogLevelDebug {
		slogLevel = slog.LevelDebug
	}
	return &Logger{
		slog: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slogLevel,
		})),
		debug:             level == config.LogLevelDebug,
		includeUnsafeCode: includeUnsafeCode,
		redactors:         buildCodeRedactors(options.Redactions),
	}
}

func (l *Logger) StdioErrorLogger() *log.Logger {
	if l == nil {
		return log.New(io.Discard, "", 0)
	}
	return log.New(&structuredLogWriter{logger: l}, "", 0)
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
		"code_size_bytes", len(record.Code),
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
		if l.includeUnsafeCode {
			sanitizedCode := redactCodeForLogging(record.Code, l.redactors)
			attrs = append(attrs,
				"code", sanitizedCode,
				"code_redacted", sanitizedCode != record.Code,
			)
		}
		if len(record.APICalls) > 0 {
			attrs = append(attrs, "api_calls", record.APICalls)
		}
		if record.StackTrace != "" {
			attrs = append(attrs, "stack_trace", record.StackTrace)
		}
	}

	l.slog.InfoContext(ctx, "tool execution", attrs...)
}

func (l *Logger) LogServerMessage(ctx context.Context, component, message string) {
	if l == nil {
		return
	}
	component = strings.TrimSpace(component)
	message = strings.TrimSpace(message)
	if component == "" {
		component = "server"
	}
	if message == "" {
		return
	}
	l.slog.WarnContext(ctx, "server message",
		"session_id", sessionIDFromContext(ctx),
		"execution_id", executionIDFromContext(ctx),
		"component", component,
		"message", message,
	)
}

type structuredLogWriter struct {
	logger *Logger
}

func (w *structuredLogWriter) Write(p []byte) (int, error) {
	if w == nil || w.logger == nil {
		return len(p), nil
	}
	message := strings.TrimSpace(string(bytes.TrimSpace(p)))
	if message != "" {
		w.logger.LogServerMessage(context.Background(), "mcp-stdio", message)
	}
	return len(p), nil
}

func CaptureStackTrace() string {
	return string(debug.Stack())
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

var defaultCodeRedactors = []codeRedactor{
	{
		pattern: regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]+=*`),
		replace: "Bearer <BEARER_TOKEN>",
	},
	{
		pattern: regexp.MustCompile(`(?i)(["']?(?:client_secret|clientSecret|[A-Z0-9_]*CLIENT_SECRET)["']?\s*[:=]\s*["'])([^"']*)(["'])`),
		replace: `${1}<CLIENT_SECRET>${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?i)(["']?(?:access_token|accessToken)["']?\s*[:=]\s*["'])([^"']*)(["'])`),
		replace: `${1}<ACCESS_TOKEN>${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?i)(["']?\b(?:api[_-]?key|apiKey|password|secret|token)\b["']?\s*[:=]\s*["'])([^"']*)(["'])`),
		replace: `${1}<REDACTED_SECRET>${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?m)\b([A-Z0-9_]*CLIENT_SECRET=)(\S+)`),
		replace: `${1}<CLIENT_SECRET>`,
	},
	{
		pattern: regexp.MustCompile(`(?m)\b([A-Z0-9_]*CLIENT_ID=)(\S+)`),
		replace: `${1}<CLIENT_ID>`,
	},
}

func buildCodeRedactors(extra []implementations.LogRedaction) []codeRedactor {
	redactors := append([]codeRedactor(nil), defaultCodeRedactors...)
	if len(extra) == 0 {
		return redactors
	}

	seen := map[string]struct{}{}
	for _, redaction := range extra {
		name := strings.TrimSpace(redaction.EnvVarName)
		if name == "" {
			continue
		}
		key := strings.ToUpper(name) + "\x00" + redaction.Placeholder
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		placeholder := strings.TrimSpace(redaction.Placeholder)
		if placeholder == "" {
			placeholder = "<REDACTED>"
		}
		quotedName := regexp.QuoteMeta(name)
		redactors = append(redactors,
			codeRedactor{
				pattern: regexp.MustCompile(`(?m)\b(` + quotedName + `=)(\S+)`),
				replace: `${1}` + placeholder,
			},
			codeRedactor{
				pattern: regexp.MustCompile(`(?i)(["']?` + quotedName + `["']?\s*[:=]\s*["'])([^"']*)(["'])`),
				replace: `${1}` + placeholder + `${3}`,
			},
		)
	}

	slices.SortStableFunc(redactors[len(defaultCodeRedactors):], func(left, right codeRedactor) int {
		switch {
		case left.replace < right.replace:
			return -1
		case left.replace > right.replace:
			return 1
		default:
			return strings.Compare(left.pattern.String(), right.pattern.String())
		}
	})
	return redactors
}

func redactCodeForLogging(code string, redactors []codeRedactor) string {
	redacted := code
	for _, redactor := range redactors {
		redacted = redactor.pattern.ReplaceAllString(redacted, redactor.replace)
	}
	return redacted
}

func sessionIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(sessionIDKey).(string)
	return value
}

func executionIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(executionIDKey).(string)
	return value
}
