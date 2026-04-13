package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mimaurer/intersight-mcp/generated"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/sandbox"
)

func TestServerToolsRegistration(t *testing.T) {
	t.Parallel()

	tools := ServerTools(stubExecutor{}, stubExecutor{}, stubExecutor{}, NewLimiter(3), 0, false, ContentMode{})
	if len(tools) != 3 {
		t.Fatalf("len(ServerTools()) = %d, want 3", len(tools))
	}

	byName := map[string]mcp.Tool{}
	for _, tool := range tools {
		byName[tool.Tool.Name] = tool.Tool
	}

	assertTool(t, byName[ToolSearch], searchTitle, true, false)
	assertTool(t, byName[ToolQuery], queryTitle, true, false)
	assertTool(t, byName[ToolMutate], mutateTitle, false, true)
	assertDescription(t, byName[ToolSearch], searchDescription)
	assertDescription(t, byName[ToolQuery], queryDescription)
	assertDescription(t, byName[ToolMutate], mutateDescription)
	assertPublicSDKOnly(t, searchDescription)
	assertPublicSDKOnly(t, queryDescription)
	assertPublicSDKOnly(t, mutateDescription)
}

func TestSchemasMatchArchitecture(t *testing.T) {
	t.Parallel()

	assertJSONEq(t, string(InputSchema()), string(inputSchemaJSON))
	assertJSONEq(t, string(MutateInputSchema()), string(mutateInputSchemaJSON))
	assertJSONEq(t, string(OutputSchema()), string(outputSchemaJSON))
}

func TestSuccessMapping(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{
		result: sandbox.Result{
			Value: map[string]any{"ok": true},
			Logs:  []string{"hello", "world"},
		},
	}, nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return { ok: true };`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false")
	}

	envelope, ok := result.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if !envelope.OK {
		t.Fatalf("envelope.OK = false, want true")
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Content[0])
	}
	if text.Text != "Success. Full result is in structuredContent. Logs: 2 line(s)." {
		t.Fatalf("unexpected success text: %q", text.Text)
	}
	if result.Meta != nil {
		t.Fatalf("unexpected result meta: %#v", result.Meta)
	}
}

func TestSuccessMappingLegacyContentMirror(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{
		result: sandbox.Result{
			Value: map[string]any{"ok": true},
			Logs:  []string{"hello", "world"},
		},
	}, nil, 0, ContentMode{MirrorStructuredContent: true})

	result, err := handler(context.Background(), toolRequest(`return { ok: true };`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Content[0])
	}
	if text.Text != "{\"ok\":true}\n\nLogs:\nhello\nworld" {
		t.Fatalf("unexpected success text: %q", text.Text)
	}
}

func TestSearchToolUsesSharedLimiter(t *testing.T) {
	t.Parallel()

	tool := NewSearchTool(stubExecutor{}, NewLimiter(1), 0, ContentMode{})
	if tool.Tool.Name != ToolSearch {
		t.Fatalf("tool name = %q, want %q", tool.Tool.Name, ToolSearch)
	}

	started := make(chan struct{}, 1)
	blocked := make(chan struct{})
	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{
		execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
			started <- struct{}{}
			<-blocked
			return sandbox.Result{Value: map[string]any{"ok": true}}, nil
		},
	}, NewLimiter(1), 0, ContentMode{})

	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		result, err := handler(context.Background(), toolRequest(`return catalog.resourceNames;`))
		if err != nil {
			t.Errorf("first handler() error = %v", err)
			return
		}
		done <- result
	}()

	<-started

	second, err := handler(context.Background(), toolRequest(`return catalog.resourceNames;`))
	if err != nil {
		t.Fatalf("second handler() error = %v", err)
	}
	if !second.IsError {
		t.Fatalf("second result.IsError = false, want true")
	}
	envelope, ok := second.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", second.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeLimit {
		t.Fatalf("error.type = %q", envelope.Error.Type)
	}
	if envelope.Error.Message != "Concurrent execution limit reached (1)" {
		t.Fatalf("error.message = %q", envelope.Error.Message)
	}

	close(blocked)
	<-done
}

func TestTelemetrySuccessMappingIgnoresPresentationHints(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeQuery, stubExecutor{
		result: sandbox.Result{
			Value:        map[string]any{"rows": 3},
			Presentation: &sandbox.PresentationHint{Kind: sandbox.PresentationKindMetricsApp},
		},
	}, nil, 0, ContentMode{})

	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolQuery,
			Arguments: map[string]any{"code": `return await sdk.telemetry.query({});`},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if result.Meta != nil {
		t.Fatalf("unexpected result meta: %#v", result.Meta)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
}

func TestErrorMapping(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeQuery, stubExecutor{
		err:    contracts.ReferenceError{Message: "spec is not defined"},
		result: sandbox.Result{Logs: []string{"trace 1"}},
	}, nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return spec;`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}

	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeReference {
		t.Fatalf("error.type = %q", envelope.Error.Type)
	}
	if envelope.Error.Hint != "The query and mutate tools do not expose spec. Use search to inspect the spec." {
		t.Fatalf("unexpected hint: %q", envelope.Error.Hint)
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "ReferenceError: spec is not defined") {
		t.Fatalf("unexpected error text: %q", text.Text)
	}
	if !strings.Contains(text.Text, "\nLogs: 1 line(s) in structuredContent.") {
		t.Fatalf("missing logs in error text: %q", text.Text)
	}
}

func TestQueryHTTPFailureNormalizesThroughMCPEnvelope(t *testing.T) {
	t.Parallel()

	exec, err := sandbox.NewQJSExecutorWithArtifacts(toolTestConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return nil, contracts.HTTPError{
				Status:  http.StatusBadGateway,
				Body:    map[string]any{"message": "downstream error"},
				Message: "Intersight returned HTTP 502",
			}
		},
	}, generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes())
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}
	handler := NewToolHandler(sandbox.ModeQuery, exec, nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return await sdk.compute.rackUnit.list();`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}

	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeHTTP {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeHTTP)
	}
	if envelope.Error.Status == nil || *envelope.Error.Status != http.StatusBadGateway {
		t.Fatalf("error.status = %#v, want %d", envelope.Error.Status, http.StatusBadGateway)
	}
	if envelope.Error.Details == nil {
		t.Fatalf("expected HTTP error details to be preserved")
	}
}

func TestQueryNetworkFailureNormalizesThroughMCPEnvelope(t *testing.T) {
	t.Parallel()

	exec, err := sandbox.NewQJSExecutorWithArtifacts(toolTestConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return nil, contracts.NetworkError{Message: "dial failed"}
		},
	}, generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes())
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}
	handler := NewToolHandler(sandbox.ModeQuery, exec, nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return await sdk.compute.rackUnit.list();`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}

	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeNetwork {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeNetwork)
	}
	if envelope.Error.Message != "dial failed" {
		t.Fatalf("error.message = %q, want %q", envelope.Error.Message, "dial failed")
	}
}

func TestQueryAuthFailureNormalizesThroughMCPEnvelope(t *testing.T) {
	t.Parallel()

	exec, err := sandbox.NewQJSExecutorWithArtifacts(toolTestConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return nil, contracts.AuthError{Message: "token refresh failed"}
		},
	}, generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes())
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}
	handler := NewToolHandler(sandbox.ModeQuery, exec, nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return await sdk.compute.rackUnit.list();`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}

	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeAuth {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeAuth)
	}
	if envelope.Error.Message != "token refresh failed" {
		t.Fatalf("error.message = %q, want %q", envelope.Error.Message, "token refresh failed")
	}
}

func TestQueryAPICallReturnsReferenceErrorEnvelope(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeQuery, sandbox.NewQJSExecutor(toolTestConfig(), stubAPICaller{}), nil, 0, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return await api.call('GET', '/api/v1/test');`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}

	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeReference {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeReference)
	}
	if envelope.Error.Message != "api is not defined" {
		t.Fatalf("unexpected error message: %q", envelope.Error.Message)
	}
}

func TestOverloadRejection(t *testing.T) {
	t.Parallel()

	blocked := make(chan struct{})
	started := make(chan struct{}, 1)
	exec := stubExecutor{
		execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
			started <- struct{}{}
			<-blocked
			return sandbox.Result{Value: map[string]any{"done": true}}, nil
		},
	}
	handler := NewToolHandler(sandbox.ModeQuery, exec, NewLimiter(1), 0, ContentMode{})

	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		result, err := handler(context.Background(), toolRequest(`return await sdk.example.widget.get({ path: { Moid: "test" } });`))
		if err != nil {
			t.Errorf("first handler() error = %v", err)
			return
		}
		done <- result
	}()

	<-started

	second, err := handler(context.Background(), toolRequest(`return await sdk.example.widget.get({ path: { Moid: "test" } });`))
	if err != nil {
		t.Fatalf("second handler() error = %v", err)
	}
	if !second.IsError {
		t.Fatalf("second result.IsError = false, want true")
	}
	envelope, ok := second.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", second.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeLimit {
		t.Fatalf("error.type = %q", envelope.Error.Type)
	}
	if envelope.Error.Message != "Concurrent execution limit reached (1)" {
		t.Fatalf("error.message = %q", envelope.Error.Message)
	}

	close(blocked)
	<-done
}

func TestMutateHandlerRequiresChangeSummary(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeMutate, stubExecutor{}, nil, 0, ContentMode{})
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolMutate,
			Arguments: map[string]any{"code": `return await sdk.ntp.policy.delete({ path: { Moid: "x" } });`},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}
	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeValidation {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeValidation)
	}
	if envelope.Error.Message != `required argument "changeSummary" not found` {
		t.Fatalf("error.message = %q", envelope.Error.Message)
	}
}

func TestMutateHandlerRejectsBlankChangeSummary(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeMutate, stubExecutor{}, nil, 0, ContentMode{})
	result, err := handler(context.Background(), mutateToolRequest("   ", `return await sdk.ntp.policy.delete({ path: { Moid: "x" } });`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}
	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeValidation {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeValidation)
	}
	if envelope.Error.Message != `required argument "changeSummary" not found` {
		t.Fatalf("error.message = %q", envelope.Error.Message)
	}
}

func TestSearchHandlerRequiresCodeThroughErrorEnvelope(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{}, nil, 0, ContentMode{})
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSearch,
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}
	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeValidation {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeValidation)
	}
	if envelope.Error.Message != `required argument "code" not found` {
		t.Fatalf("error.message = %q", envelope.Error.Message)
	}
}

func TestMutateHandlerPassesOnlyCodeToExecutor(t *testing.T) {
	t.Parallel()

	var (
		gotCode string
		gotMode sandbox.Mode
	)
	handler := NewToolHandler(sandbox.ModeMutate, stubExecutor{
		execute: func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
			gotCode = code
			gotMode = mode
			return sandbox.Result{Value: map[string]any{"ok": true}}, nil
		},
	}, nil, 0, ContentMode{})

	result, err := handler(context.Background(), mutateToolRequest("Delete the NTP policy", `return await sdk.ntp.policy.delete({ path: { Moid: "x" } });`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false")
	}
	if gotMode != sandbox.ModeMutate {
		t.Fatalf("mode = %q, want %q", gotMode, sandbox.ModeMutate)
	}
	if gotCode != `return await sdk.ntp.policy.delete({ path: { Moid: "x" } });` {
		t.Fatalf("code = %q", gotCode)
	}
}

func TestToolHandlerDoesNotRecountDuplicatedMCPEnvelopeSize(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{
		result: sandbox.Result{
			Value: map[string]any{"data": strings.Repeat("a", 80)},
			Logs:  []string{"trace"},
		},
	}, nil, 120, ContentMode{})

	result, err := handler(context.Background(), toolRequest(`return { ok: true };`))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false")
	}
}

func assertTool(t *testing.T, tool mcp.Tool, title string, readOnly, destructive bool) {
	t.Helper()

	if tool.Name == "" {
		t.Fatalf("tool name is empty")
	}
	if tool.Annotations.Title != title {
		t.Fatalf("%s title = %q, want %q", tool.Name, tool.Annotations.Title, title)
	}
	if tool.Annotations.ReadOnlyHint == nil || *tool.Annotations.ReadOnlyHint != readOnly {
		t.Fatalf("%s readOnly = %#v, want %v", tool.Name, tool.Annotations.ReadOnlyHint, readOnly)
	}
	if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != destructive {
		t.Fatalf("%s destructive = %#v, want %v", tool.Name, tool.Annotations.DestructiveHint, destructive)
	}
	wantInputSchema := string(inputSchemaJSON)
	if tool.Name == ToolMutate {
		wantInputSchema = string(mutateInputSchemaJSON)
	}
	assertJSONEq(t, string(tool.RawInputSchema), wantInputSchema)
	assertJSONEq(t, string(tool.RawOutputSchema), string(outputSchemaJSON))
	if tool.Name == ToolQuery && tool.Meta != nil {
		t.Fatalf("query tool meta = %#v, want nil by default", tool.Meta)
	}
}

func assertDescription(t *testing.T, tool mcp.Tool, want string) {
	t.Helper()

	if tool.Description != want {
		t.Fatalf("%s description mismatch", tool.Name)
	}
}

func assertPublicSDKOnly(t *testing.T, description string) {
	t.Helper()

	forbidden := []string{
		"api.call(",
		"dryRun",
		"dry-run",
	}
	for _, token := range forbidden {
		if strings.Contains(description, token) {
			t.Fatalf("public description unexpectedly contains %q", token)
		}
	}
}

func assertJSONEq(t *testing.T, got, want string) {
	t.Helper()

	var gotJSON any
	if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
		t.Fatalf("unmarshal got JSON: %v", err)
	}
	var wantJSON any
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatalf("unmarshal want JSON: %v", err)
	}
	gotBytes, _ := json.Marshal(gotJSON)
	wantBytes, _ := json.Marshal(wantJSON)
	if string(gotBytes) != string(wantBytes) {
		t.Fatalf("JSON mismatch\ngot:  %s\nwant: %s", gotBytes, wantBytes)
	}
}

func toolRequest(code string) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSearch,
			Arguments: map[string]any{"code": code},
		},
	}
}

func mutateToolRequest(changeSummary, code string) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: ToolMutate,
			Arguments: map[string]any{
				"changeSummary": changeSummary,
				"code":          code,
			},
		},
	}
}

type stubExecutor struct {
	result  sandbox.Result
	err     error
	execute func(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error)
}

func (s stubExecutor) Execute(ctx context.Context, code string, mode sandbox.Mode) (sandbox.Result, error) {
	if s.execute != nil {
		return s.execute(ctx, code, mode)
	}
	return s.result, s.err
}

func (s stubExecutor) Close() error { return nil }

var _ sandbox.Executor = stubExecutor{}
var _ mcpserver.ToolHandlerFunc = NewToolHandler(sandbox.ModeSearch, stubExecutor{}, nil, 0, ContentMode{})

type stubAPICaller struct {
	do func(ctx context.Context, operation contracts.OperationDescriptor) (any, error)
}

func (s stubAPICaller) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if s.do != nil {
		return s.do(ctx, operation)
	}
	return map[string]any{"method": operation.Method, "path": operation.Path}, nil
}

func TestToolErrorResultHandlesNil(t *testing.T) {
	t.Parallel()

	result := toolErrorResult(nil, nil, ContentMode{})
	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeInternal {
		t.Fatalf("error.type = %q", envelope.Error.Type)
	}
}

func TestToolHandlerPropagatesBindErrors(t *testing.T) {
	t.Parallel()

	handler := NewToolHandler(sandbox.ModeSearch, stubExecutor{}, nil, 0, ContentMode{})
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ToolSearch,
			Arguments: []any{"not-an-object"},
		},
	})
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true")
	}
	envelope, ok := result.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("StructuredContent type = %T", result.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeValidation {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeValidation)
	}
	if !strings.Contains(envelope.Error.Message, "cannot unmarshal") {
		t.Fatalf("unexpected error message: %q", envelope.Error.Message)
	}
}

func TestToolDescriptionTokenBudget(t *testing.T) {
	t.Parallel()

	const maxApproxTokens = 2300

	descriptions := map[string]string{
		ToolSearch: searchDescription,
		ToolQuery:  queryDescription,
		ToolMutate: mutateDescription,
	}

	total := 0
	for name, description := range descriptions {
		approx := approximateTokenCount(description)
		t.Logf("%s description approx tokens: %d", name, approx)
		if approx == 0 {
			t.Fatalf("%s description token count = 0", name)
		}
		total += approx
	}

	t.Logf("total tool description approx tokens: %d", total)
	if total > maxApproxTokens {
		t.Fatalf("tool descriptions total approx tokens = %d, want <= %d; trim examples before changing the architecture budget", total, maxApproxTokens)
	}
}

func approximateTokenCount(text string) int {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r == '_' || r == '$':
			return false
		default:
			return true
		}
	})

	total := 0
	for _, field := range fields {
		total += approxFieldTokens(field)
	}
	return total
}

func approxFieldTokens(field string) int {
	if field == "" {
		return 0
	}
	if len(field) <= 4 {
		return 1
	}
	return (len(field) + 3) / 4
}

func TestToolDescriptionTokenBudgetReport(t *testing.T) {
	t.Parallel()

	total := approximateTokenCount(searchDescription) +
		approximateTokenCount(queryDescription) +
		approximateTokenCount(mutateDescription)
	if total <= 0 {
		t.Fatal("expected positive token estimate")
	}
	t.Log(fmt.Sprintf("approximate total tool description tokens: %d", total))
}

func toolTestConfig() sandbox.Config {
	cfg := sandbox.DefaultConfig()
	cfg.GlobalTimeout = 5 * time.Second
	cfg.PerCallTimeout = 500 * time.Millisecond
	return cfg
}
