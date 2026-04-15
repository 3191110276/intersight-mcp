package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/server"
)

func TestEmbeddedArtifactsValidate(t *testing.T) {
	t.Parallel()

	artifacts := app.Target.Artifacts()
	if err := server.ValidateEmbeddedArtifacts(artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog); err != nil {
		t.Fatalf("ValidateEmbeddedArtifacts() error = %v", err)
	}
}

func TestServeStartsWithoutCredentialsForOfflineSearch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := app.ServeWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, nil); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
}

func TestServeVerificationSmoke(t *testing.T) {
	t.Parallel()

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.ServeWithIO(ctx, nil, stdinReader, stdoutWriter, &bytes.Buffer{}, nil)
		_ = stdoutWriter.Close()
	}()

	lineCh := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(stdoutReader)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				lineCh <- line
			}
		}
		close(lineCh)
	}()

	writeJSONLine(t, stdinWriter, initializeRequest())
	writeJSONLine(t, stdinWriter, toolsListRequest(2))
	writeJSONLine(t, stdinWriter, toolCallRequest(3, "search", `
const resource = catalog.resources["administered.identitiesme"];
const schema = resource ? catalog.schema(resource.schema) : null;
return {
  hasCatalog: !!resource,
  hasSchemaHelper: typeof catalog.schema === "function",
  hasPathLookup: !!catalog.paths["/api/v1/administered/identities/me"],
  hasSchema: !!schema
};
`))
	writeJSONLine(t, stdinWriter, toolCallRequest(4, "query", `
return await sdk.administered.identitiesme.list();
`))

	lines := make([]string, 0, 4)
	for len(lines) < 4 {
		select {
		case line, ok := <-lineCh:
			if !ok {
				t.Fatalf("stdout closed after %d responses, want 4", len(lines))
			}
			lines = append(lines, line)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for MCP responses: %v", ctx.Err())
		}
	}
	_ = stdinWriter.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ServeWithIO() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for ServeWithIO to return: %v", ctx.Err())
	}

	responses := indexResponsesByID(t, lines)

	assertNoRPCError(t, responses[1])

	tools := decodeToolsListResult(t, responses[2])
	if len(tools) != 3 {
		t.Fatalf("tool count = %d, want 3", len(tools))
	}
	for _, name := range []string{"search", "query", "mutate"} {
		if !tools[name] {
			t.Fatalf("missing tool %q in tools/list response", name)
		}
	}

	searchResult := decodeToolResult(t, responses[3])
	if searchResult.IsError {
		t.Fatalf("search returned error: %#v", searchResult.StructuredContent)
	}
	searchEnvelope, ok := searchResult.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected search envelope type: %T", searchResult.StructuredContent)
	}
	searchPayload := mustMap(t, searchEnvelope.Result)
	if searchPayload["hasCatalog"] != true || searchPayload["hasSchemaHelper"] != true || searchPayload["hasPathLookup"] != true || searchPayload["hasSchema"] != true {
		t.Fatalf("unexpected search result: %#v", searchPayload)
	}

	queryResult := decodeToolResult(t, responses[4])
	if !queryResult.IsError {
		t.Fatalf("query without API key IsError = false, want true")
	}
	queryEnvelope, ok := queryResult.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("unexpected query envelope type: %T", queryResult.StructuredContent)
	}
	if queryEnvelope.Error.Type != contracts.ErrorTypeAuth {
		t.Fatalf("unexpected query error type: %q", queryEnvelope.Error.Type)
	}
	if !strings.Contains(queryEnvelope.Error.Message, "Meraki API key is not configured") {
		t.Fatalf("unexpected query error message: %q", queryEnvelope.Error.Message)
	}
}

func TestServeReadOnlyOmitsMutateTool(t *testing.T) {
	t.Parallel()

	stdinReader, stdinWriter := io.Pipe()
	var stdout bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		writeJSONLine(t, stdinWriter, initializeRequest())
		writeJSONLine(t, stdinWriter, toolsListRequest(2))
		_ = stdinWriter.Close()
	}()

	if err := app.ServeWithIO(ctx, []string{"--read-only"}, stdinReader, &stdout, &bytes.Buffer{}, nil); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}

	lines := splitLines(stdout.String())
	if len(lines) != 2 {
		t.Fatalf("response count = %d, want 2", len(lines))
	}

	responses := indexResponsesByID(t, lines)
	tools := decodeToolsListResult(t, responses[2])
	if len(tools) != 2 {
		t.Fatalf("tool count = %d, want 2", len(tools))
	}
	if !tools["search"] || !tools["query"] {
		t.Fatalf("unexpected tools list: %#v", tools)
	}
	if tools["mutate"] {
		t.Fatalf("mutate tool should be omitted in read-only mode")
	}
}

func initializeRequest() map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "verification-client",
				"version": "1.0.0",
			},
		},
	}
}

func toolsListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
		"params":  map[string]any{},
	}
}

func toolCallRequest(id int, name, code string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": map[string]any{"code": code},
		},
	}
}

func writeJSONLine(t *testing.T, w io.Writer, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}

func splitLines(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func assertNoRPCError(t *testing.T, line string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %#v", payload["error"])
	}
}

func decodeToolsListResult(t *testing.T, line string) map[string]bool {
	t.Helper()
	var payload struct {
		Error  any `json:"error"`
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %#v", payload.Error)
	}
	out := make(map[string]bool, len(payload.Result.Tools))
	for _, tool := range payload.Result.Tools {
		out[tool.Name] = true
	}
	return out
}

func decodeToolResult(t *testing.T, line string) toolResponse {
	t.Helper()
	var payload struct {
		Error  any          `json:"error"`
		Result toolResponse `json:"result"`
	}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %#v", payload.Error)
	}
	return payload.Result
}

func indexResponsesByID(t *testing.T, lines []string) map[int]string {
	t.Helper()
	out := make(map[int]string, len(lines))
	for _, line := range lines {
		var payload struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		out[payload.ID] = line
	}
	return out
}

type toolResponse struct {
	IsError           bool `json:"isError"`
	StructuredContent any  `json:"structuredContent"`
}

func (r *toolResponse) UnmarshalJSON(data []byte) error {
	type rawToolResponse struct {
		IsError           bool            `json:"isError"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	var raw rawToolResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.IsError = raw.IsError
	if raw.IsError {
		var envelope contracts.ErrorEnvelope
		if err := json.Unmarshal(raw.StructuredContent, &envelope); err != nil {
			return err
		}
		r.StructuredContent = envelope
		return nil
	}
	var envelope contracts.SuccessEnvelope
	if err := json.Unmarshal(raw.StructuredContent, &envelope); err != nil {
		return err
	}
	r.StructuredContent = envelope
	return nil
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value type = %T, want map[string]any", value)
	}
	return m
}
