package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/internal/config"
)

func TestLoggerInfoLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelInfo, false, LoggerOptions{})
	ctx := WithExecutionID(WithSessionID(context.Background(), "session-1"), "exec-1")
	logger.LogExecution(ctx, ExecutionRecord{
		Tool:            "search",
		Code:            "return 1",
		ChangeSummary:   "",
		Duration:        25 * time.Millisecond,
		APICallCount:    0,
		ResultSizeBytes: 64,
		Success:         true,
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["session_id"] != "session-1" || payload["execution_id"] != "exec-1" {
		t.Fatalf("missing execution metadata: %#v", payload)
	}
	if payload["code"] != nil {
		t.Fatalf("did not expect full code in info logs: %#v", payload)
	}
	if payload["code_hash"] == "" {
		t.Fatalf("expected code hash: %#v", payload)
	}
	if payload["code_size_bytes"] != float64(len("return 1")) {
		t.Fatalf("expected code_size_bytes in logs: %#v", payload)
	}
	if payload["change_summary"] != "" {
		t.Fatalf("expected empty change_summary: %#v", payload)
	}
}

func TestLoggerDebugLevelOmitsCodeUnlessEnabled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, false, LoggerOptions{})
	logger.LogExecution(context.Background(), ExecutionRecord{
		Tool:            "query",
		Code:            "return await api.call('GET', '/api/v1/x')",
		ChangeSummary:   "Inspect x",
		Duration:        100 * time.Millisecond,
		APICallCount:    1,
		ResultSizeBytes: 128,
		Success:         false,
		ErrorType:       "TimeoutError",
		ErrorMessage:    "Request timeout (15s)",
		APICalls: []APICallRecord{{
			Method:       "GET",
			Path:         "/api/v1/x",
			Status:       504,
			ResponseSize: 0,
			DurationMS:   15_000,
		}},
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["code"] != nil {
		t.Fatalf("did not expect debug log to include full code by default: %#v", payload)
	}
	if payload["api_calls"] == nil {
		t.Fatalf("expected debug log to include API call details: %#v", payload)
	}
	if payload["change_summary"] != "Inspect x" {
		t.Fatalf("expected change_summary in logs: %#v", payload)
	}
}

func TestLoggerDebugLevelIncludesRedactedCodeWhenEnabled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, true, LoggerOptions{})
	logger.LogExecution(context.Background(), ExecutionRecord{
		Tool:          "query",
		Code:          `return { token: "secret-token", Authorization: "Bearer abc.def.ghi", client_secret: "super-secret" }`,
		Duration:      10 * time.Millisecond,
		Success:       true,
		APICallCount:  0,
		ChangeSummary: "",
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	code, ok := payload["code"].(string)
	if !ok {
		t.Fatalf("expected debug log to include redacted code when enabled: %#v", payload)
	}
	if strings.Contains(code, "super-secret") || strings.Contains(code, "secret-token") || strings.Contains(code, "abc.def.ghi") {
		t.Fatalf("expected secret values to be redacted: %q", code)
	}
	if !strings.Contains(code, "<CLIENT_SECRET>") || !strings.Contains(code, "<REDACTED_SECRET>") || !strings.Contains(code, "Bearer <BEARER_TOKEN>") {
		t.Fatalf("expected redaction markers in code: %q", code)
	}
	if payload["code_redacted"] != true {
		t.Fatalf("expected code_redacted=true: %#v", payload)
	}
}

func TestAPICallRecorderSnapshot(t *testing.T) {
	t.Parallel()

	ctx, recorder := WithAPICallRecorder(context.Background())
	RecordAPICall(ctx, APICallRecord{
		Method:       "GET",
		Path:         "/api/v1/test",
		Status:       200,
		ResponseSize: 42,
		DurationMS:   12,
	})

	calls := recorder.Snapshot()
	if len(calls) != 1 {
		t.Fatalf("len(Snapshot()) = %d, want 1", len(calls))
	}
	if calls[0].Path != "/api/v1/test" {
		t.Fatalf("calls[0].Path = %q, want /api/v1/test", calls[0].Path)
	}
}

func TestStdioErrorLoggerRoutesToStructuredLogs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, false, LoggerOptions{})

	if _, err := io.WriteString(logger.StdioErrorLogger().Writer(), "Error reading input: boom\n"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["component"] != "mcp-stdio" {
		t.Fatalf("component = %#v, want %q", payload["component"], "mcp-stdio")
	}
	if payload["msg"] != "server message" {
		t.Fatalf("msg = %#v, want %q", payload["msg"], "server message")
	}
	if payload["message"] != "Error reading input: boom" {
		t.Fatalf("message = %#v, want %q", payload["message"], "Error reading input: boom")
	}
}

func TestLoggerDebugLevelIncludesProviderConfiguredEnvRedactions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, true, LoggerOptions{
		Redactions: []implementations.LogRedaction{
			{EnvVarName: "ACME_ACCESS_KEY", Placeholder: "<ACCESS_KEY>"},
		},
	})
	logger.LogExecution(context.Background(), ExecutionRecord{
		Tool:          "query",
		Code:          `return { env: "ACME_ACCESS_KEY=top-secret", ACME_ACCESS_KEY: "top-secret" }`,
		Duration:      10 * time.Millisecond,
		Success:       true,
		APICallCount:  0,
		ChangeSummary: "",
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	code, ok := payload["code"].(string)
	if !ok {
		t.Fatalf("expected debug log to include redacted code when enabled: %#v", payload)
	}
	if strings.Contains(code, "top-secret") {
		t.Fatalf("expected provider-configured secret value to be redacted: %q", code)
	}
	if !strings.Contains(code, "<ACCESS_KEY>") {
		t.Fatalf("expected provider-configured redaction marker in code: %q", code)
	}
}
