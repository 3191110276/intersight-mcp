package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/config"
)

func TestLoggerInfoLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelInfo, false)
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
	if payload["change_summary"] != "" {
		t.Fatalf("expected empty change_summary: %#v", payload)
	}
}

func TestLoggerDebugLevelOmitsCodeUnlessEnabled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, false)
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

func TestLoggerDebugLevelIncludesCodeWhenEnabled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewLogger(&buf, config.LogLevelDebug, true)
	logger.LogExecution(context.Background(), ExecutionRecord{
		Tool:          "query",
		Code:          "return await sdk.compute.rackUnit.list()",
		Duration:      10 * time.Millisecond,
		Success:       true,
		APICallCount:  0,
		ChangeSummary: "",
	})

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["code"] == nil {
		t.Fatalf("expected debug log to include full code when enabled: %#v", payload)
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
	logger := NewLogger(&buf, config.LogLevelDebug, false)

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
