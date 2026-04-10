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
	"github.com/mimaurer/intersight-mcp/generated"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

func TestServeWithIOVerificationMatrix(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeIntersight(t)
	defer fake.Close()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=" + fake.URL(),
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveWithIO(ctx, nil, stdinReader, stdoutWriter, &bytes.Buffer{}, env, generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes(), generated.SearchCatalogBytes())
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
return {
  hasCatalog: !!catalog.resources["compute.rackUnit"],
  hasMetrics: !!catalog.metrics.byName["system.cpu.utilization_user"],
  hasCollection: !!spec.paths["/api/v1/compute/RackUnits"]?.get,
  hasObject: !!spec.paths["/api/v1/compute/RackUnits/{Moid}"]?.get,
  hasSchema: !!spec.schemas["compute.RackUnit"]
};
`))
	writeJSONLine(t, stdinWriter, toolCallRequest(4, "query", `
const collection = await sdk.compute.rackUnit.list();
const first = await sdk.compute.rackUnit.get({ path: { Moid: collection.Results[0].Moid } });
return {
  count: collection.Count,
  firstName: first.Name,
  firstMoid: first.Moid
};
`))
	writeJSONLine(t, stdinWriter, toolCallRequest(5, "query", `
return await api.call("POST", "/api/v1/ntp/Policies", { body: { Name: "should-not-work" } });
`))
	writeJSONLine(t, stdinWriter, toolCallRequest(6, "query", `
return await sdk.ntp.policy.create({
  body: {
    Enabled: true,
    Name: "ntp-policy-preview",
    Timezone: "UTC",
    NtpServers: ["pool.ntp.org"]
  }
});
`))
	writeJSONLine(t, stdinWriter, mutateToolCallRequest(7, "Create NTP policy ntp-policy-01 in org-1", `
return await sdk.ntp.policy.create({
  body: {
    Enabled: true,
    Timezone: "UTC",
    NtpServers: ["pool.ntp.org"]
  }
});
`))
	lines := make([]string, 0, 7)
	for len(lines) < 7 {
		select {
		case line, ok := <-lineCh:
			if !ok {
				t.Fatalf("stdout closed after %d responses, want 7", len(lines))
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
			t.Fatalf("serveWithIO() error = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for serveWithIO to return: %v", ctx.Err())
	}

	if len(lines) != 7 {
		t.Fatalf("response count = %d, want 7", len(lines))
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

	search := decodeToolResult(t, responses[3])
	if search.IsError {
		t.Fatalf("search returned error: %#v", search.StructuredContent)
	}
	searchEnvelope, ok := search.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected search envelope type: %T", search.StructuredContent)
	}
	searchResult := mustMap(t, searchEnvelope.Result)
	if searchResult["hasCatalog"] != true || searchResult["hasCollection"] != true || searchResult["hasObject"] != true || searchResult["hasSchema"] != true {
		t.Fatalf("unexpected search result: %#v", searchResult)
	}

	query := decodeToolResult(t, responses[4])
	if query.IsError {
		t.Fatalf("query returned error: %#v", query.StructuredContent)
	}
	queryEnvelope, ok := query.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected query envelope type: %T", query.StructuredContent)
	}
	queryResult := mustMap(t, queryEnvelope.Result)
	if queryResult["count"] != float64(2) {
		t.Fatalf("unexpected query count: %#v", queryResult["count"])
	}
	if queryResult["firstName"] != "rack-alpha" {
		t.Fatalf("unexpected query firstName: %#v", queryResult["firstName"])
	}

	rejected := decodeToolResult(t, responses[5])
	if !rejected.IsError {
		t.Fatalf("query write rejection IsError = false, want true")
	}
	rejectedEnvelope, ok := rejected.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("unexpected rejection envelope type: %T", rejected.StructuredContent)
	}
	if rejectedEnvelope.Error.Type != contracts.ErrorTypeReference {
		t.Fatalf("unexpected rejection error type: %q", rejectedEnvelope.Error.Type)
	}
	if rejectedEnvelope.Error.Message != "api is not defined" {
		t.Fatalf("unexpected rejection error message: %q", rejectedEnvelope.Error.Message)
	}

	writeValidation := decodeToolResult(t, responses[6])
	if writeValidation.IsError {
		t.Fatalf("query write validation returned error: %#v", writeValidation.StructuredContent)
	}
	writeValidationEnvelope, ok := writeValidation.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected query write envelope type: %T", writeValidation.StructuredContent)
	}
	writeValidationResult := mustMap(t, writeValidationEnvelope.Result)
	if writeValidationResult["valid"] != true {
		t.Fatalf("unexpected query write valid: %#v", writeValidationResult["valid"])
	}

	mutate := decodeToolResult(t, responses[7])
	if mutate.IsError {
		t.Fatalf("mutate returned error: %#v", mutate.StructuredContent)
	}
	mutateEnvelope, ok := mutate.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected mutate envelope type: %T", mutate.StructuredContent)
	}
	mutateResult := mustMap(t, mutateEnvelope.Result)
	if mutateResult["Moid"] != "policy-1" {
		t.Fatalf("unexpected mutate Moid: %#v", mutateResult["Moid"])
	}

	policy := fake.LastCreatedPolicy()
	if policy == nil {
		t.Fatalf("expected fake server to record created policy")
	}
	if policy["Enabled"] != true || policy["Timezone"] != "UTC" {
		t.Fatalf("unexpected recorded policy: %#v", policy)
	}
}

var verificationTestSpec = []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "paths": {
    "/api/v1/compute/RackUnits": {
      "get": {
        "summary": "List rack units",
        "operationId": "GetComputeRackUnitList"
      }
    },
    "/api/v1/compute/RackUnits/{moid}": {
      "get": {
        "summary": "Get a rack unit",
        "operationId": "GetComputeRackUnit"
      }
    },
    "/api/v1/ntp/Policies": {
      "post": {
        "summary": "Create an NTP policy",
        "operationId": "CreateNtpPolicy",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "Name": { "type": "string" },
                  "Enabled": { "type": "boolean" },
                  "Timezone": { "type": "string" },
                  "NtpServers": {
                    "type": "array",
                    "items": { "type": "string" }
                  },
                  "AuthenticatedNtpServers": {
                    "type": "array",
                    "items": { "type": "string" }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "schemas": {
    "compute.RackUnit": {
      "type": "object",
      "properties": {
        "Moid": { "type": "string" },
        "Name": { "type": "string" }
      }
    },
    "ntp.Policy": {
      "type": "object",
      "properties": {
        "Name": { "type": "string" },
        "Enabled": { "type": "boolean" },
        "Timezone": { "type": "string" },
        "NtpServers": {
          "type": "array",
          "items": { "type": "string" }
        },
        "AuthenticatedNtpServers": {
          "type": "array",
          "items": { "type": "string" }
        }
      }
    }
  },
  "tags": []
}`)

var verificationTestCatalog = []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {
    "compute.rackUnit.get": {
      "sdkMethod": "compute.rackUnit.get",
      "summary": "Get a rack unit",
      "resource": "compute.RackUnit",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "GetComputeRackUnit",
        "method": "GET",
        "pathTemplate": "/api/v1/compute/RackUnits/{moid}",
        "path": "/api/v1/compute/RackUnits/{moid}",
        "responseMode": "json",
        "validationPlan": {
          "kind": "none"
        },
        "followUpPlan": {
          "kind": "none"
        }
      }
    },
    "compute.rackUnit.list": {
      "sdkMethod": "compute.rackUnit.list",
      "summary": "List rack units",
      "resource": "compute.RackUnit",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "GetComputeRackUnitList",
        "method": "GET",
        "pathTemplate": "/api/v1/compute/RackUnits",
        "path": "/api/v1/compute/RackUnits",
        "responseMode": "json",
        "validationPlan": {
          "kind": "none"
        },
        "followUpPlan": {
          "kind": "none"
        }
      }
    },
    "ntp.policy.create": {
      "sdkMethod": "ntp.policy.create",
      "summary": "Create an NTP policy",
      "resource": "ntp.Policy",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "CreateNtpPolicy",
        "method": "POST",
        "pathTemplate": "/api/v1/ntp/Policies",
        "path": "/api/v1/ntp/Policies",
        "responseMode": "json",
        "validationPlan": {
          "kind": "none"
        },
        "followUpPlan": {
          "kind": "none"
        }
      },
      "requestBodyRequired": true,
      "requestBodyFields": [
        "AuthenticatedNtpServers",
        "Enabled",
        "Name",
        "NtpServers",
        "Timezone"
      ]
    }
  }
}`)

var verificationTestRules = []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {
    "ntp.policy.create": {
      "sdkMethod": "ntp.policy.create",
      "operationId": "CreateNtpPolicy",
      "resource": "ntp.Policy",
      "rules": [
        {
          "kind": "required",
          "require": [
            { "field": "Enabled" }
          ]
        },
        {
          "kind": "required",
          "require": [
            { "field": "Timezone" }
          ]
        },
        {
          "kind": "one_of",
          "requireAny": [
            { "field": "NtpServers" },
            { "field": "AuthenticatedNtpServers" }
          ]
        }
      ]
    }
  }
}`)

var verificationTestSearchCatalog = []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "resources": {
    "compute.rackUnit": {
      "schema": "compute.RackUnit",
      "path": "/api/v1/compute/RackUnits/{moid?}",
      "operations": ["get", "list"]
    },
    "ntp.policy": {
      "schema": "ntp.Policy",
      "path": "/api/v1/ntp/Policies",
      "operations": ["create"]
    }
  },
  "resourceNames": ["compute.rackUnit", "ntp.policy"],
  "paths": {
    "/api/v1/compute/RackUnits": ["compute.rackUnit"],
    "/api/v1/compute/rackunits": ["compute.rackUnit"],
    "/compute/RackUnits": ["compute.rackUnit"],
    "/compute/rackunits": ["compute.rackUnit"],
    "/api/v1/compute/RackUnits/{moid}": ["compute.rackUnit"],
    "/api/v1/compute/rackunits/{moid}": ["compute.rackUnit"],
    "/compute/RackUnits/{moid}": ["compute.rackUnit"],
    "/compute/rackunits/{moid}": ["compute.rackUnit"],
    "/api/v1/ntp/Policies": ["ntp.policy"],
    "/api/v1/ntp/policies": ["ntp.policy"],
    "/ntp/Policies": ["ntp.policy"],
    "/ntp/policies": ["ntp.policy"]
  }
}`)

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

func mutateToolCallRequest(id int, changeSummary, code string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "mutate",
			"arguments": map[string]any{
				"changeSummary": changeSummary,
				"code":          code,
			},
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
