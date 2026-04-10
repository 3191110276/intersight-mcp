package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/config"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
	"github.com/mimaurer/intersight-mcp/intersight"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func TestNewRuntimeRegistersExactlyThreeTools(t *testing.T) {
	t.Parallel()

	rt, err := NewRuntime(RuntimeConfig{
		SearchExecutor: stubExecutor{},
		QueryExecutor:  stubExecutor{},
		MutateExecutor: stubExecutor{},
		MaxConcurrent:  3,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer rt.Close()

	tools := rt.MCPServer().ListTools()
	if len(tools) != 3 {
		t.Fatalf("len(ListTools()) = %d, want 3", len(tools))
	}
	for _, name := range []string{"search", "query", "mutate"} {
		if tools[name] == nil {
			t.Fatalf("missing tool %q", name)
		}
	}
}

func TestRuntimeSuccessfulStdioStartup(t *testing.T) {
	t.Parallel()

	rt, err := NewRuntime(RuntimeConfig{
		SearchExecutor: stubExecutor{},
		QueryExecutor:  stubExecutor{},
		MutateExecutor: stubExecutor{},
		MaxConcurrent:  2,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer rt.Close()

	var stdin bytes.Buffer
	stdin.WriteString(mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}))
	stdin.WriteString(mustJSONLine(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}))

	var stdout bytes.Buffer
	if err := rt.Listen(context.Background(), &stdin, &stdout); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	lines := splitLines(stdout.String())
	if len(lines) != 2 {
		t.Fatalf("response count = %d, want 2", len(lines))
	}

	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}
	if initResp["error"] != nil {
		t.Fatalf("unexpected initialize error: %#v", initResp["error"])
	}

	var toolsResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools/list response: %v", err)
	}
	if len(toolsResp.Result.Tools) != 3 {
		t.Fatalf("tool count = %d, want 3", len(toolsResp.Result.Tools))
	}
}

func TestRuntimeShutdownCancelsInflightExecutionOnStdinClose(t *testing.T) {
	t.Parallel()

	rt, err := NewRuntime(RuntimeConfig{
		SearchExecutor: stubExecutor{},
		QueryExecutor: stubExecutor{
			execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
				<-ctx.Done()
				return sandbox.Result{}, contracts.InternalError{Message: "execution canceled", Err: ctx.Err()}
			},
		},
		MutateExecutor: stubExecutor{},
		MaxConcurrent:  1,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer rt.Close()

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()

	errCh := make(chan error, 1)
	responsesCh := make(chan []string, 1)
	go func() {
		errCh <- rt.Listen(context.Background(), stdinReader, stdoutWriter)
		_ = stdoutWriter.Close()
	}()

	go func() {
		scanner := bufio.NewScanner(stdoutReader)
		responses := make([]string, 0, 2)
		for scanner.Scan() {
			responses = append(responses, scanner.Text())
		}
		responsesCh <- responses
	}()

	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	})
	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "query",
			"arguments": map[string]any{
				"code": `return await sdk.example.widget.get({ path: { Moid: "test" } });`,
			},
		},
	})
	_ = stdinWriter.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Listen() did not return after stdin closed")
	}

	var responses []string
	select {
	case responses = <-responsesCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for stdout responses")
	}

	if len(responses) < 2 {
		t.Fatalf("response count = %d, want at least 2", len(responses))
	}

	var toolResp struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent contracts.ErrorEnvelope `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(responses[len(responses)-1]), &toolResp); err != nil {
		t.Fatalf("unmarshal tool response: %v", err)
	}
	if !toolResp.Result.IsError {
		t.Fatalf("tool response IsError = false, want true")
	}
	if toolResp.Result.StructuredContent.Error.Type != contracts.ErrorTypeInternal {
		t.Fatalf("error.type = %q, want %q", toolResp.Result.StructuredContent.Error.Type, contracts.ErrorTypeInternal)
	}
}

func TestRuntimeShutdownCancelsInflightExecutionOnContextCancel(t *testing.T) {
	t.Parallel()

	rt, err := NewRuntime(RuntimeConfig{
		SearchExecutor: stubExecutor{},
		QueryExecutor: stubExecutor{
			execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
				<-ctx.Done()
				return sandbox.Result{}, contracts.InternalError{Message: "execution canceled", Err: ctx.Err()}
			},
		},
		MutateExecutor: stubExecutor{},
		MaxConcurrent:  1,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer rt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	stdinReader, stdinWriter := io.Pipe()
	var stdout bytes.Buffer

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.Listen(ctx, stdinReader, &stdout)
	}()

	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	})
	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "query",
			"arguments": map[string]any{
				"code": `return await sdk.example.widget.get({ path: { Moid: "test" } });`,
			},
		},
	})

	cancel()
	_ = stdinWriter.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Listen() did not return after context cancellation")
	}

	responses := splitLines(stdout.String())
	if len(responses) == 0 {
		t.Fatal("expected at least one response before shutdown")
	}
}

func TestRuntimeShutdownCancelsInflightHTTPCallOnContextCancel(t *testing.T) {
	t.Parallel()

	requestStarted := make(chan struct{}, 1)
	requestCanceled := make(chan struct{}, 1)
	api := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/test":
			requestStarted <- struct{}{}
			<-r.Context().Done()
			requestCanceled <- struct{}{}
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	client := intersight.NewClient(api.Client(), api.URL, staticTokenProvider("test-token"))
	rt, err := NewRuntime(RuntimeConfig{
		SearchExecutor: stubExecutor{},
		QueryExecutor: stubExecutor{
			execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
				_, err := client.DoJSON(ctx, http.MethodGet, "/api/v1/test", intersight.RequestOptions{})
				return sandbox.Result{}, err
			},
		},
		MutateExecutor: stubExecutor{},
		MaxConcurrent:  1,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer rt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	stdinReader, stdinWriter := io.Pipe()
	var stdout bytes.Buffer

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.Listen(ctx, stdinReader, &stdout)
	}()

	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	})
	writeJSONLine(t, stdinWriter, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "query",
			"arguments": map[string]any{
				"code": `return await sdk.example.widget.get({ path: { Moid: "test" } });`,
			},
		},
	})

	select {
	case <-requestStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for HTTP request to start")
	}

	cancel()
	_ = stdinWriter.Close()

	select {
	case <-requestCanceled:
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight HTTP request was not canceled")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Listen() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Listen() did not return after context cancellation")
	}
}

func TestWrapToolHandlerLogsOverloadRejection(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := internalpkg.NewLogger(&buf, config.LogLevelDebug)
	handler := wrapToolHandler("query", func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		time.Sleep(10 * time.Millisecond)
		return &mcp.CallToolResult{
			IsError:           true,
			StructuredContent: contracts.NormalizeError(contracts.LimitError{Message: "Concurrent execution limit reached (1)"}, nil),
		}, nil
	}, logger)

	_, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "query",
			Arguments: map[string]any{"code": `return 1;`},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["execution_id"] == "" {
		t.Fatalf("execution_id missing from log: %#v", payload)
	}
	if payload["error_type"] != contracts.ErrorTypeLimit {
		t.Fatalf("error_type = %v, want %q", payload["error_type"], contracts.ErrorTypeLimit)
	}
	if payload["duration_ms"] == nil {
		t.Fatalf("duration_ms missing from log: %#v", payload)
	}
}

func TestWrapToolHandlerLogsChangeSummary(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := internalpkg.NewLogger(&buf, config.LogLevelDebug)
	handler := wrapToolHandler("mutate", func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			IsError:           false,
			StructuredContent: contracts.Success(map[string]any{"ok": true}, nil),
		}, nil
	}, logger)

	_, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "mutate",
			Arguments: map[string]any{
				"changeSummary": "Delete policy x",
				"code":          `return 1;`,
			},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if payload["change_summary"] != "Delete policy x" {
		t.Fatalf("change_summary = %#v, want %q", payload["change_summary"], "Delete policy x")
	}
}

func mustJSONLine(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(append(data, '\n'))
}

func writeJSONLine(t *testing.T, w io.Writer, value any) {
	t.Helper()
	if _, err := io.WriteString(w, mustJSONLine(t, value)); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
}

func splitLines(s string) []string {
	raw := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

type stubExecutor struct {
	execute func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error)
}

func (s stubExecutor) Execute(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
	if s.execute != nil {
		return s.execute(ctx, code, mode)
	}
	return sandbox.Result{Value: map[string]any{"ok": true}}, nil
}

func (s stubExecutor) Close() error { return nil }

var _ sandbox.Executor = stubExecutor{}

type staticTokenProvider string

func (s staticTokenProvider) Token(context.Context) (string, error) {
	return string(s), nil
}
