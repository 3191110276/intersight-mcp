package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	targetintersight "github.com/mimaurer/intersight-mcp/implementations/intersight"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func newTestQJSExecutorWithArtifacts(cfg Config, client APICaller, specJSON, catalogJSON, rulesJSON []byte) (Executor, error) {
	return NewQJSExecutorWithArtifactsAndExtensions(cfg, client, specJSON, catalogJSON, rulesJSON, targetintersight.SandboxExtensions())
}

func TestSearchWrongGlobalReferenceError(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(testConfig(), artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	_, err = exec.Execute(context.Background(), `return await api.call('GET', '/api/v1/test');`, ModeSearch)
	if err == nil {
		t.Fatalf("expected error")
	}

	var refErr contracts.ReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected ReferenceError, got %T", err)
	}
}

func TestQueryWrongGlobalReferenceError(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules)
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}
	_, err = exec.Execute(context.Background(), `return spec.paths;`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var refErr contracts.ReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected ReferenceError, got %T", err)
	}
}

func TestNewQJSExecutorSupportsSDKQueries(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			if operation.Method != http.MethodGet {
				t.Fatalf("operation.Method = %q, want %q", operation.Method, http.MethodGet)
			}
			if operation.Path != "/api/v1/compute/RackUnits" {
				t.Fatalf("operation.Path = %q", operation.Path)
			}
			return map[string]any{"Results": []any{map[string]any{"Moid": "rack-1"}}}, nil
		},
	}, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules)
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}

result, err := exec.Execute(context.Background(), `return await sdk.compute.rackUnits.list();`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	results, ok := value["Results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("unexpected result.Value = %#v", result.Value)
	}
}

func TestSearchDiscoveryGlobalsAvailable(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(testConfig(), artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
const resource = catalog.resources["compute.rackUnits"];
return {
  catalogResources: Object.keys(catalog.resources || {}).length,
  catalogNames: Object.keys(catalog.resourceNames || {}).length,
  catalogMetricGroups: Object.keys((catalog.metrics && catalog.metrics.groups) || {}).length,
  hasSchemaHelper: typeof catalog.schema === "function",
  schemaName: resource ? resource.schema : null,
  schemaType: resource ? catalog.schema(resource.schema)?.type ?? null : null,
  legacySpecType: typeof spec,
  legacySDKType: typeof sdk,
  legacyRulesType: typeof rules
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if value["catalogResources"] == nil || value["catalogNames"] == nil || value["catalogMetricGroups"] == nil {
		t.Fatalf("unexpected search discovery payload: %#v", result.Value)
	}
	if value["hasSchemaHelper"] != true || value["schemaName"] == nil || value["schemaType"] == nil {
		t.Fatalf("expected schema drilldown payload: %#v", result.Value)
	}
	if value["legacySpecType"] != "undefined" || value["legacySDKType"] != "undefined" || value["legacyRulesType"] != "undefined" {
		t.Fatalf("expected legacy globals to be undefined: %#v", result.Value)
	}
}

func TestExecutorsFromBundleProvideExpectedGlobals(t *testing.T) {
	bundle, err := LoadArtifactBundleWithExtensions(
		targetintersight.Artifacts().ResolvedSpec,
		targetintersight.Artifacts().SDKCatalog,
		targetintersight.Artifacts().Rules,
		targetintersight.Artifacts().SearchCatalog,
		targetintersight.SandboxExtensions(),
	)
	if err != nil {
		t.Fatalf("LoadArtifactBundle() error = %v", err)
	}

	searchExec, err := NewSearchExecutorFromBundle(testConfig(), bundle)
	if err != nil {
		t.Fatalf("NewSearchExecutorFromBundle() error = %v", err)
	}
	defer searchExec.Close()

	searchResult, err := searchExec.Execute(context.Background(), `
return {
  catalogResources: Object.keys(catalog.resources || {}).length,
  catalogMetricGroups: Object.keys((catalog.metrics && catalog.metrics.groups) || {}).length
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("search Execute() error = %v", err)
	}
	searchValue, ok := searchResult.Value.(map[string]any)
	if !ok {
		t.Fatalf("search result.Value type = %T", searchResult.Value)
	}
	if searchValue["catalogResources"] == nil || searchValue["catalogMetricGroups"] == nil {
		t.Fatalf("unexpected search payload: %#v", searchResult.Value)
	}

	queryExec, err := NewQJSExecutorFromBundle(testConfig(), stubAPICaller{}, bundle)
	if err != nil {
		t.Fatalf("NewQJSExecutorFromBundle() error = %v", err)
	}

	queryResult, err := queryExec.Execute(context.Background(), `
return await sdk.ntp.policies.create({
  body: {
    Name: "ntp-policy-01",
    Enabled: true,
    Timezone: "UTC",
    NtpServers: ["pool.ntp.org"],
    Organization: { Moid: "5ddf1d456972652d30bc0a10" }
  }
});
`, ModeQuery)
	if err != nil {
		t.Fatalf("query Execute() error = %v", err)
	}
	queryValue, ok := queryResult.Value.(map[string]any)
	if !ok {
		t.Fatalf("query result.Value type = %T", queryResult.Value)
	}
	if valid, _ := queryValue["valid"].(bool); !valid {
		t.Fatalf("unexpected validation report: %#v", queryResult.Value)
	}
}

func TestSearchMetricsCatalogAvailable(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(testConfig(), artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
return {
  metricKeys: Object.keys((catalog.metrics && catalog.metrics.byName) || {}),
  groupKeys: Object.keys((catalog.metrics && catalog.metrics.groups) || {}),
  hasExamples: !!(catalog.metrics && catalog.metrics.examples),
  metric: catalog.metrics.byName["system.cpu.utilization_user"],
  group: catalog.metrics.groups["system.cpu"]
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if value["metric"] == nil || value["group"] == nil {
		t.Fatalf("unexpected metrics payload: %#v", value)
	}
	if value["metricKeys"] == nil || value["groupKeys"] == nil {
		t.Fatalf("unexpected metric key payloads: %#v", value)
	}
	metric, ok := value["metric"].(map[string]any)
	if !ok {
		t.Fatalf("metric payload type = %T", value["metric"])
	}
	dimensions, ok := metric["dimensions"].([]any)
	if !ok || len(dimensions) == 0 {
		t.Fatalf("metric dimensions = %#v, want inherited queryable dimensions", metric["dimensions"])
	}
	if value["hasExamples"] != false {
		t.Fatalf("expected metrics.examples to be hidden from catalog: %#v", value)
	}
}

func TestSearchCatalogSchemaLookup(t *testing.T) {
	t.Parallel()

	exec, err := NewSearchExecutor(testConfig(), []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules), []byte(testSearchCatalog))
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
const resource = catalog.resources["example.widget"];
return {
  schemaName: resource?.schema ?? null,
  schema: resource ? catalog.schema(resource.schema) : null,
  missing: catalog.schema("missing.Schema"),
  hasMetrics: typeof catalog.metrics !== "undefined"
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if value["schemaName"] != "example.Widget" {
		t.Fatalf("schemaName = %#v, want example.Widget", value["schemaName"])
	}
	schema, ok := value["schema"].(map[string]any)
	if !ok {
		t.Fatalf("schema payload type = %T", value["schema"])
	}
	if schema["type"] != "object" {
		t.Fatalf("schema.type = %#v, want object", schema["type"])
	}
	if _, ok := value["missing"]; ok && value["missing"] != nil {
		t.Fatalf("missing schema = %#v, want nil/undefined", value["missing"])
	}
	if value["hasMetrics"] != false {
		t.Fatalf("expected metrics to be omitted for catalogs without metrics: %#v", value)
	}
}

func TestConsoleLogCapture(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(testConfig(), artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
console.log("hello from search");
return { ok: true };
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Logs) == 0 {
		t.Fatalf("expected captured logs")
	}
	if result.Logs[0] != "hello from search" {
		t.Fatalf("unexpected logs: %#v", result.Logs)
	}
}

func TestConsoleLogsCountTowardOutputLimit(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.MaxOutputBytes = 64

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(cfg, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	_, err = exec.Execute(context.Background(), `
console.log("abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz");
return { ok: true };
`, ModeSearch)
	if err == nil {
		t.Fatalf("expected error")
	}

	var tooLarge contracts.OutputTooLarge
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected OutputTooLarge, got %T", err)
	}
}

func TestConsoleLogsAreTruncatedToOutputLimit(t *testing.T) {
	t.Parallel()

	buf := newLogBuffer(12)
	if _, err := buf.Write([]byte("hello world again")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	lines := buf.Lines()
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2 (%#v)", len(lines), lines)
	}
	if lines[0] != "hello world " {
		t.Fatalf("unexpected first log line: %#v", lines[0])
	}
	if lines[1] != "[logs truncated to fit output limit]" {
		t.Fatalf("unexpected truncation marker: %#v", lines[1])
	}
}

func TestQueryAPICallIsReferenceError(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}
	_, err = exec.Execute(context.Background(), `return await api.call('POST', '/api/v1/test');`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var refErr contracts.ReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected ReferenceError, got %T", err)
	}
	if !strings.Contains(refErr.Error(), "api is not defined") {
		t.Fatalf("unexpected error: %v", refErr)
	}
}

func TestQueryAPICallOptionsNoLongerMatter(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}
	_, err = exec.Execute(context.Background(), `return await api.call('POST', '/api/v1/test', { dryRun: 'yes' });`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var refErr contracts.ReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected ReferenceError, got %T", err)
	}
}

const testSDKSpec = `{
  "paths": {
    "/api/v1/example/Widgets": {
      "get": {
        "operationId": "GetExampleWidgetList",
        "parameters": [
          { "name": "$select", "in": "query", "required": false, "schema": { "type": "string" } }
        ]
      },
      "post": {
        "operationId": "CreateExampleWidget",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["ClassId", "ObjectType", "Name", "Mode"],
                "properties": {
                  "ClassId": { "type": "string", "enum": ["example.Widget"] },
                  "Moid": { "type": "string" },
                  "Name": { "type": "string" },
                  "Mode": { "type": "string", "enum": ["fast", "safe"] },
                  "ObjectType": { "type": "string", "enum": ["example.Widget"] },
                  "Organization": {
                    "type": "object",
                    "$expandTarget": "organization.Organization",
                    "x-relationship": true,
                    "x-relationshipTarget": "organization.Organization",
                    "x-writeForms": ["moidRef", "typedMoRef"]
                  }
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/example/Widgets/{Moid}": {
      "get": {
        "operationId": "GetExampleWidgetByMoid",
        "parameters": [
          { "name": "Moid", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "$select", "in": "query", "required": false, "schema": { "type": "string" } },
          { "name": "$top", "in": "query", "required": false, "schema": { "type": "integer", "format": "int32" } },
          { "name": "$count", "in": "query", "required": false, "schema": { "type": "boolean" } }
        ]
      },
      "patch": {
        "operationId": "PatchExampleWidget",
        "parameters": [
          { "name": "Moid", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["Name", "Mode"],
                "properties": {
                  "Moid": { "type": "string" },
                  "Name": { "type": "string" },
                  "Mode": { "type": "string", "enum": ["fast", "safe"] }
                }
              }
            }
          }
        }
      }
    }
  },
  "schemas": {
    "example.Widget": {
      "type": "object",
      "properties": {
        "Moid": { "type": "string" },
        "Name": { "type": "string" },
        "Mode": { "type": "string", "enum": ["fast", "safe"] }
      }
    }
  }
}`

const testSDKCatalog = `{
  "methods": {
    "example.widget.list": {
      "sdkMethod": "example.widget.list",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "GetExampleWidgetList",
        "method": "GET",
        "pathTemplate": "/api/v1/example/Widgets",
        "path": "/api/v1/example/Widgets",
        "responseMode": "json",
        "validationPlan": { "kind": "none" },
        "followUpPlan": { "kind": "none" }
      },
      "queryParameters": ["$select"]
    },
    "example.widget.get": {
      "sdkMethod": "example.widget.get",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "GetExampleWidgetByMoid",
        "method": "GET",
        "pathTemplate": "/api/v1/example/Widgets/{Moid}",
        "path": "/api/v1/example/Widgets/{Moid}",
        "responseMode": "json",
        "validationPlan": { "kind": "none" },
        "followUpPlan": { "kind": "none" }
      },
      "pathParameters": ["Moid"],
      "queryParameters": ["$count", "$select", "$top"]
    },
    "example.widget.create": {
      "sdkMethod": "example.widget.create",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "CreateExampleWidget",
        "method": "POST",
        "pathTemplate": "/api/v1/example/Widgets",
        "path": "/api/v1/example/Widgets",
        "responseMode": "json",
        "validationPlan": { "kind": "none" },
        "followUpPlan": { "kind": "none" }
      },
      "requestBodyRequired": true
    },
    "example.widget.update": {
      "sdkMethod": "example.widget.update",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "PatchExampleWidget",
        "method": "PATCH",
        "pathTemplate": "/api/v1/example/Widgets/{Moid}",
        "path": "/api/v1/example/Widgets/{Moid}",
        "responseMode": "json",
        "validationPlan": { "kind": "none" },
        "followUpPlan": { "kind": "none" }
      },
      "pathParameters": ["Moid"],
      "requestBodyRequired": true
    }
  }
}`

const testSDKRules = `{
  "methods": {}
}`

const testSemanticRules = `{
  "methods": {
    "example.widget.create": {
      "sdkMethod": "example.widget.create",
      "operationId": "CreateExampleWidget",
      "resource": "example.Widget",
      "rules": [
        {
          "kind": "conditional",
          "when": { "field": "Mode", "equals": "fast" },
          "require": [
            { "field": "Organization", "target": "organization.Organization" }
          ]
        }
      ]
    }
  }
}`

const testSearchCatalog = `{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "resources": {
    "example.widget": {
      "schema": "example.Widget",
      "createFields": {
        "Mode": { "type": "string", "enum": true },
        "Name": { "type": "string", "required": true },
        "Organization": {
          "ref": "organization.Organization",
          "example": { "Moid": "<organization-moid>" }
        }
      },
      "rules": [
        {
          "kind": "conditional",
          "when": { "field": "Mode", "equals": "fast" },
          "require": [
            { "field": "Organization", "target": "organization.Organization" }
          ]
        }
      ],
      "operations": ["create", "list"]
    }
  },
  "resourceNames": ["example.widget"],
  "paths": {
    "/api/v1/example/Widgets": ["example.widget"]
  }
}`

const testSearchCatalogWithPostUpdate = `{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "resources": {
    "example.widget": {
      "schema": "example.Widget",
      "createFields": {
        "Mode": { "type": "string", "enum": true },
        "Name": { "type": "string" }
      },
      "operations": ["post", "update"]
    }
  },
  "resourceNames": ["example.widget"],
  "paths": {
    "/api/v1/example/Widgets/{Moid}": ["example.widget"]
  }
}`

func TestSearchCatalogHidesOperationMetadata(t *testing.T) {
	t.Parallel()

	exec, err := NewSearchExecutor(testConfig(), []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules), []byte(testSearchCatalog))
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
const resource = catalog.resources["example.widget"];
return {
  catalogSchema: resource.schema ?? null,
  catalogCreateFields: resource.createFields ?? null,
  catalogRules: resource.rules ?? null,
  catalogOperations: resource.operations ?? null
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if got := value["catalogSchema"]; got != "example.Widget" {
		t.Fatalf("catalog schema = %#v, want example.Widget", got)
	}
	if got := value["catalogCreateFields"]; got == nil {
		t.Fatalf("catalog createFields = %#v, want create-focused fields retained", got)
	}
	fields, ok := value["catalogCreateFields"].(map[string]any)
	if !ok {
		t.Fatalf("catalog createFields type = %T", value["catalogCreateFields"])
	}
	if _, exists := fields["Moid"]; exists {
		t.Fatalf("catalog createFields = %#v, want readOnly fields removed", fields)
	}
	if got := value["catalogRules"]; got == nil {
		t.Fatalf("catalog rules = %#v, want resource-level rules retained", got)
	}
	operations, ok := value["catalogOperations"].([]any)
	if !ok {
		t.Fatalf("catalog operations type = %T", value["catalogOperations"])
	}
	if got := fmt.Sprint(operations); got != "[create list]" {
		t.Fatalf("catalog operations = %v, want [create list]", operations)
	}
}

func TestSearchCatalogHidesPostWhenUpdateExists(t *testing.T) {
	t.Parallel()

	exec, err := NewSearchExecutor(testConfig(), []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules), []byte(testSearchCatalogWithPostUpdate))
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
const resource = catalog.resources["example.widget"];
return {
  hasPost: resource.operations.includes("post"),
  hasUpdate: resource.operations.includes("update")
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if got := value["hasPost"]; got != false {
		t.Fatalf("hasPost = %#v, want false", got)
	}
	if got := value["hasUpdate"]; got != true {
		t.Fatalf("hasUpdate = %#v, want true", got)
	}
}

func TestSearchCatalogLazyProxiesSupportEnumerationAndLookup(t *testing.T) {
	t.Parallel()

	exec, err := NewSearchExecutor(testConfig(), []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules), []byte(testSearchCatalog))
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
const resourceKeys = Object.keys(catalog.resources || {});
const pathKeys = Object.keys(catalog.paths || {});
return {
  resourceKeys,
  pathKeys,
  resourceSchema: catalog.resources["example.widget"]?.schema ?? null,
  pathLookup: catalog.paths["/api/v1/example/Widgets"] ?? null
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if got := fmt.Sprint(value["resourceKeys"]); got != "[example.widget]" {
		t.Fatalf("resourceKeys = %v, want [example.widget]", value["resourceKeys"])
	}
	if got := fmt.Sprint(value["pathKeys"]); got != "[/api/v1/example/Widgets]" {
		t.Fatalf("pathKeys = %v, want [/api/v1/example/Widgets]", value["pathKeys"])
	}
	if got := value["resourceSchema"]; got != "example.Widget" {
		t.Fatalf("resourceSchema = %#v, want example.Widget", got)
	}
	if got := fmt.Sprint(value["pathLookup"]); got != "[example.widget]" {
		t.Fatalf("pathLookup = %v, want [example.widget]", value["pathLookup"])
	}
}

func TestSearchDoesNotExposeRawDiscoveryGlobals(t *testing.T) {
	t.Parallel()

	exec, err := NewSearchExecutor(testConfig(), []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules), []byte(testSearchCatalog))
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	result, err := exec.Execute(context.Background(), `
return {
  hasSpec: typeof spec !== "undefined",
  hasSDK: typeof sdk !== "undefined",
  hasRules: typeof rules !== "undefined"
};
`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	value, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("result.Value type = %T", result.Value)
	}
	if got := value["hasSpec"]; got != false {
		t.Fatalf("hasSpec = %#v, want false", got)
	}
	if got := value["hasSDK"]; got != false {
		t.Fatalf("hasSDK = %#v, want false", got)
	}
	if got := value["hasRules"]; got != false {
		t.Fatalf("hasRules = %#v, want false", got)
	}
}

func TestQuerySDKReadCompilesOperationDescriptor(t *testing.T) {
	t.Parallel()

	var captured contracts.OperationDescriptor
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			captured = operation
			return map[string]any{"ok": true}, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.get({
  path: { Moid: 'widget-1' },
  query: { '$select': 'Name,Mode', '$top': 10, '$count': true }
});
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.OperationID != "GetExampleWidgetByMoid" {
		t.Fatalf("operationId = %q", captured.OperationID)
	}
	if captured.Method != http.MethodGet {
		t.Fatalf("method = %q", captured.Method)
	}
	if captured.PathTemplate != "/api/v1/example/Widgets/{Moid}" {
		t.Fatalf("pathTemplate = %q", captured.PathTemplate)
	}
	if captured.Path != "/api/v1/example/Widgets/widget-1" {
		t.Fatalf("path = %q", captured.Path)
	}
	if got := captured.PathParams["Moid"]; got != "widget-1" {
		t.Fatalf("path param Moid = %q", got)
	}
	if got := captured.QueryParams["$select"][0]; got != "Name,Mode" {
		t.Fatalf("query $select = %q", got)
	}
	if got := captured.QueryParams["$top"][0]; got != "10" {
		t.Fatalf("query $top = %q", got)
	}
	if got := captured.QueryParams["$count"][0]; got != "true" {
		t.Fatalf("query $count = %q", got)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok || payload["ok"] != true {
		t.Fatalf("unexpected result payload: %#v", result.Value)
	}
}

func TestQuerySDKReturnsOfflineValidationReportForWriteOperation(t *testing.T) {
	t.Parallel()

	calls := 0
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			calls++
			return nil, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: 'fast',
    Organization: { Moid: 'org-1' }
  }
});
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("query made %d API calls for write validation, want 0", calls)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != true {
		t.Fatalf("valid = %#v, want true", payload["valid"])
	}
	issues := validationIssuesFromPayload(t, payload)
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want empty", issues)
	}
	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, true)
	assertLayer(t, layers, "openapi_request_schema", true, true)
	assertLayer(t, layers, "rules_semantic", true, true)
}

func TestQueryCustomTelemetryQueryPostsDruidBody(t *testing.T) {
	t.Parallel()

	var captured contracts.OperationDescriptor
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			captured = operation
			return map[string]any{"rows": 3}, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: ['switchId', 'portId'],
  intervals: ['2026-04-01/2026-04-09'],
  granularity: 'hour',
  aggregations: [
    { type: 'longSum', name: 'totalPackets', fieldName: 'packets' }
  ]
});
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.OperationID != "CustomTelemetryQuery" {
		t.Fatalf("operationId = %q", captured.OperationID)
	}
	if captured.Method != http.MethodPost {
		t.Fatalf("method = %q", captured.Method)
	}
	if captured.Path != "/api/v1/telemetry/TimeSeries" {
		t.Fatalf("path = %q", captured.Path)
	}
	body, ok := captured.Body.(map[string]any)
	if !ok {
		t.Fatalf("body type = %T", captured.Body)
	}
	if got := body["queryType"]; got != "groupBy" {
		t.Fatalf("queryType = %#v", got)
	}
	if got := body["dataSource"]; got != "fabric_port" {
		t.Fatalf("dataSource = %#v", got)
	}
	dimensions, ok := body["dimensions"].([]any)
	if !ok || len(dimensions) != 2 {
		t.Fatalf("dimensions = %#v", body["dimensions"])
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result payload: %#v", result.Value)
	}
	if result.Presentation != nil {
		t.Fatalf("presentation = %#v, want nil by default", result.Presentation)
	}
	switch got := payload["rows"].(type) {
	case int:
		if got != 3 {
			t.Fatalf("rows = %v, want 3", got)
		}
	case int64:
		if got != 3 {
			t.Fatalf("rows = %v, want 3", got)
		}
	case float64:
		if got != 3 {
			t.Fatalf("rows = %v, want 3", got)
		}
	default:
		t.Fatalf("rows type = %T, want numeric 3", payload["rows"])
	}
}

func TestQueryCustomTelemetryQueryRequiresQueryModeAndBody(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `return await sdk.telemetry.query({});`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `requires dataSource`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: ['switchId'],
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09']
});
`, ModeMutate)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `only runs in query`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestQueryCustomTelemetryQueryRejectsExplicitImageRenderMode(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: ['switchId'],
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09'],
  render: 'image'
});
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `render must be one of`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestQueryCustomTelemetryQueryRejectsUnknownRenderMode(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: ['switchId'],
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09'],
  render: 'app'
});
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `render must be one of`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestQueryCustomTelemetryQueryRejectsAppRenderModeEvenWhenEnabledInternally(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.EnableMetricsApps = true

	exec, err := newTestQJSExecutorWithArtifacts(cfg, stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: ['switchId'],
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09'],
  render: 'app'
});
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `render must be one of`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestQueryCustomTelemetryQueryRequiresGroupByFields(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09']
});
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}
	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `requires dimensions`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.telemetry.query({
  dataSource: 'fabric_port',
  dimensions: 'switchId',
  granularity: 'hour',
  intervals: ['2026-04-01/2026-04-09']
});
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), `dimensions must be an array`) {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestQuerySDKReturnsSchemaFailureReportForWriteOperation(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: false
  }
});
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != false {
		t.Fatalf("valid = %#v, want false", payload["valid"])
	}

	issues := validationIssuesFromPayload(t, payload)
	if len(issues) == 0 {
		t.Fatalf("issues = %#v, want non-empty array", payload["issues"])
	}
	if issues[0]["source"] != validationSourceOpenAPI {
		t.Fatalf("source = %#v, want %q", issues[0]["source"], validationSourceOpenAPI)
	}
	if issues[0]["type"] != "enum" {
		t.Fatalf("type = %#v, want enum", issues[0]["type"])
	}
}

func TestValidateSDKReturnsOfflineValidationReport(t *testing.T) {
	t.Parallel()

	calls := 0
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			calls++
			return nil, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: 'fast',
    Organization: { Moid: 'org-1' }
  }
});
`, ModeValidate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("validate made %d API calls, want 0", calls)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != true {
		t.Fatalf("valid = %#v, want true", payload["valid"])
	}
	issues := validationIssuesFromPayload(t, payload)
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want empty", issues)
	}
	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, true)
	assertLayer(t, layers, "openapi_request_schema", true, true)
	assertLayer(t, layers, "rules_semantic", true, true)
}

func TestValidateSDKReturnsSchemaFailureReport(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: false
  }
});
`, ModeValidate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != false {
		t.Fatalf("valid = %#v, want false", payload["valid"])
	}

	issues := validationIssuesFromPayload(t, payload)
	if len(issues) == 0 {
		t.Fatalf("issues = %#v, want non-empty array", payload["issues"])
	}
	if issues[0]["source"] != validationSourceOpenAPI {
		t.Fatalf("source = %#v, want %q", issues[0]["source"], validationSourceOpenAPI)
	}
	if issues[0]["type"] != "enum" {
		t.Fatalf("type = %#v, want enum", issues[0]["type"])
	}

	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, true)
	assertLayer(t, layers, "openapi_request_schema", true, false)
	assertLayer(t, layers, "rules_semantic", true, true)
}

func TestValidateSDKReturnsRulesFailureReport(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: 'fast'
  }
});
`, ModeValidate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != false {
		t.Fatalf("valid = %#v, want false", payload["valid"])
	}

	issues := validationIssuesFromPayload(t, payload)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want 1 issue", issues)
	}
	if issues[0]["source"] != validationSourceRules {
		t.Fatalf("source = %#v, want %q", issues[0]["source"], validationSourceRules)
	}
	if issues[0]["type"] != "required" {
		t.Fatalf("type = %#v, want required", issues[0]["type"])
	}

	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, true)
	assertLayer(t, layers, "openapi_request_schema", true, true)
	assertLayer(t, layers, "rules_semantic", true, false)
}

func TestValidateSDKReturnsMixedFailureReport(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Mode: 'fast'
  }
});
`, ModeValidate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}

	issues := validationIssuesFromPayload(t, payload)
	if len(issues) != 2 {
		t.Fatalf("issues = %#v, want 2 issues", issues)
	}
	sources := map[string]bool{}
	for _, issue := range issues {
		source, _ := issue["source"].(string)
		sources[source] = true
	}
	if !sources[validationSourceOpenAPI] || !sources[validationSourceRules] {
		t.Fatalf("sources = %#v, want openapi and rules", sources)
	}

	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, true)
	assertLayer(t, layers, "openapi_request_schema", true, false)
	assertLayer(t, layers, "rules_semantic", true, false)
}

func TestValidateSDKReturnsSDKContractFailureReport(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  query: "bad"
});
`, ModeValidate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["valid"] != false {
		t.Fatalf("valid = %#v, want false", payload["valid"])
	}

	issues := validationIssuesFromPayload(t, payload)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want 1 issue", issues)
	}
	if issues[0]["source"] != validationSourceSDKContract {
		t.Fatalf("source = %#v, want %q", issues[0]["source"], validationSourceSDKContract)
	}
	if issues[0]["type"] != "type_mismatch" {
		t.Fatalf("type = %#v, want type_mismatch", issues[0]["type"])
	}

	layers := validationLayersFromPayload(t, payload)
	assertLayer(t, layers, "sdk_contract", true, false)
	assertLayer(t, layers, "openapi_request_schema", false, true)
	assertLayer(t, layers, "rules_semantic", false, true)
}

func TestValidateSDKRejectsReadOperation(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.example.widget.get({
  path: { Moid: 'widget-1' }
});
`, ModeValidate)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), "should run as a normal query") {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func validationIssuesFromPayload(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()

	raw, ok := payload["issues"].([]any)
	if !ok {
		t.Fatalf("issues type = %T", payload["issues"])
	}
	out := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("issue type = %T", entry)
		}
		out = append(out, item)
	}
	return out
}

func validationLayersFromPayload(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()

	raw, ok := payload["layers"].([]any)
	if !ok {
		t.Fatalf("layers type = %T", payload["layers"])
	}
	out := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("layer type = %T", entry)
		}
		out = append(out, item)
	}
	return out
}

func assertLayer(t *testing.T, layers []map[string]any, name string, ran, passed bool) {
	t.Helper()

	for _, layer := range layers {
		if layer["name"] != name {
			continue
		}
		if layer["ran"] != ran || layer["passed"] != passed {
			t.Fatalf("layer %q = %#v, want ran=%v passed=%v", name, layer, ran, passed)
		}
		return
	}
	t.Fatalf("missing layer %q in %#v", name, layers)
}

func TestValidateRejectsAPICall(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `return await api.call('GET', '/api/v1/test');`, ModeValidate)
	if err == nil {
		t.Fatalf("expected error")
	}

	var refErr contracts.ReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected ReferenceError, got %T", err)
	}
	if !strings.Contains(refErr.Error(), "api is not defined") {
		t.Fatalf("unexpected error: %v", refErr)
	}
}

func TestMutateSDKNormalizesRelationshipPayload(t *testing.T) {
	t.Parallel()

	var captured contracts.OperationDescriptor
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			captured = operation
			return operation.Body, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: 'fast',
    Organization: { Moid: 'org-1' }
  }
});
`, ModeMutate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body, ok := captured.Body.(map[string]any)
	if !ok {
		t.Fatalf("unexpected body type: %T", captured.Body)
	}
	if body["ClassId"] != "example.Widget" {
		t.Fatalf("ClassId = %#v", body["ClassId"])
	}
	if body["ObjectType"] != "example.Widget" {
		t.Fatalf("ObjectType = %#v", body["ObjectType"])
	}
	organization, ok := body["Organization"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected relationship payload: %#v", body["Organization"])
	}
	if organization["ClassId"] != "mo.MoRef" {
		t.Fatalf("ClassId = %#v", organization["ClassId"])
	}
	if organization["ObjectType"] != "organization.Organization" {
		t.Fatalf("ObjectType = %#v", organization["ObjectType"])
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	normalized, _ := payload["Organization"].(map[string]any)
	if normalized["ClassId"] != "mo.MoRef" {
		t.Fatalf("normalized ClassId = %#v", normalized["ClassId"])
	}
}

func TestMutateSDKLogsSemanticRuleViolationAndContinues(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSemanticRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
return await sdk.example.widget.create({
  body: {
    Name: 'widget-a',
    Mode: 'fast'
  }
});
`, ModeMutate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Logs) < 2 {
		t.Fatalf("logs = %#v, want warning lines", result.Logs)
	}
	if !strings.Contains(result.Logs[0], "mutate continued") {
		t.Fatalf("unexpected warning summary: %#v", result.Logs)
	}
	if !strings.Contains(result.Logs[1], `"sdkMethod":"example.widget.create"`) {
		t.Fatalf("unexpected warning payload: %#v", result.Logs)
	}
}

func TestMutateSDKRejectsPathBodyMoidMismatch(t *testing.T) {
	t.Parallel()

	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.example.widget.update({
  path: { Moid: 'widget-1' },
  body: {
    Moid: 'widget-2',
    Name: 'widget-a',
    Mode: 'fast'
  }
});
`, ModeMutate)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), "does not match body Moid") {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestMutateServerProfileRejectsTargetPlatformMismatchInPolicyBucket(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	var operations []contracts.OperationDescriptor
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			operations = append(operations, operation)
			switch {
			case operation.Method == http.MethodGet && operation.Path == "/api/v1/firmware/Policies/fw-1":
				return map[string]any{
					"Moid":           "fw-1",
					"ObjectType":     "firmware.Policy",
					"TargetPlatform": "UnifiedEdgeServer",
				}, nil
			default:
				t.Fatalf("unexpected operation: %s %s", operation.Method, operation.Path)
				return nil, nil
			}
		},
	}, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules)
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.server.profiles.create({
  body: {
    Name: 'profile-a',
    Type: 'instance',
    TargetPlatform: 'Standalone',
    Organization: { Moid: 'org-1' },
    PolicyBucket: [
      { Moid: 'fw-1', ObjectType: 'firmware.Policy' }
    ]
  }
});
`, ModeMutate)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), "local validation") {
		t.Fatalf("unexpected error: %v", validationErr)
	}
	if len(operations) != 1 || operations[0].Method != http.MethodGet {
		t.Fatalf("operations = %#v, want single preflight GET", operations)
	}
}

func TestMutateServerProfileRejectsPersistedPolicyBucketMismatch(t *testing.T) {
	t.Parallel()

	artifacts := targetintersight.Artifacts()
	var operations []contracts.OperationDescriptor
	exec, err := newTestQJSExecutorWithArtifacts(testConfig(), stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			operations = append(operations, operation)
			switch {
			case operation.Method == http.MethodGet && operation.Path == "/api/v1/firmware/Policies/fw-1":
				return map[string]any{
					"Moid":           "fw-1",
					"ObjectType":     "firmware.Policy",
					"TargetPlatform": "Standalone",
				}, nil
			case operation.Method == http.MethodPost && operation.Path == "/api/v1/server/Profiles":
				return map[string]any{
					"Moid":       "profile-1",
					"ObjectType": "server.Profile",
				}, nil
			case operation.Method == http.MethodGet && operation.Path == "/api/v1/server/Profiles/profile-1":
				return map[string]any{
					"Moid":         "profile-1",
					"ObjectType":   "server.Profile",
					"PolicyBucket": []any{},
				}, nil
			default:
				t.Fatalf("unexpected operation: %s %s", operation.Method, operation.Path)
				return nil, nil
			}
		},
	}, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules)
	if err != nil {
		t.Fatalf("newTestQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
return await sdk.server.profiles.create({
  body: {
    Name: 'profile-a',
    Type: 'instance',
    TargetPlatform: 'Standalone',
    Organization: { Moid: 'org-1' },
    PolicyBucket: [
      { Moid: 'fw-1', ObjectType: 'firmware.Policy' }
    ]
  }
});
`, ModeMutate)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), "local validation") {
		t.Fatalf("unexpected error: %v", validationErr)
	}
	if len(operations) != 3 {
		t.Fatalf("operations = %#v, want preflight GET + POST + verification GET", operations)
	}
}

func TestPerCallTimeout(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.PerCallTimeout = 2 * time.Second
	cfg.GlobalTimeout = 5 * time.Second

	exec, err := newTestQJSExecutorWithArtifacts(cfg, stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return nil, contracts.TimeoutError{Message: "Intersight request failed", Err: ctx.Err()}
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
try {
  await sdk.example.widget.get({ path: { Moid: 'widget-1' } });
  return { ok: false };
} catch (err) {
  return err;
}
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["kind"] != "timeout" {
		t.Fatalf("unexpected timeout kind: %#v", payload["kind"])
	}
	if payload["message"] != "Request timeout (2s)" {
		t.Fatalf("unexpected timeout message: %#v", payload["message"])
	}
	if payload["timeoutSeconds"] != int64(2) {
		t.Fatalf("unexpected timeout seconds: %#v", payload["timeoutSeconds"])
	}
}

func TestPerCallTimeoutUncaughtNormalizesToTimeoutError(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.PerCallTimeout = 2 * time.Second
	cfg.GlobalTimeout = 5 * time.Second

	exec, err := newTestQJSExecutorWithArtifacts(cfg, stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return nil, contracts.TimeoutError{Message: "Intersight request failed", Err: ctx.Err()}
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `return await sdk.example.widget.get({ path: { Moid: 'widget-1' } });`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T (%v)", err, err)
	}
}

func TestGlobalTimeout(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.SearchTimeout = 40 * time.Millisecond

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(cfg, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	_, err = exec.Execute(context.Background(), `while (true) {}`, ModeSearch)
	if err == nil {
		t.Fatalf("expected error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T (%v)", err, err)
	}
}

func TestSearchTimeoutCoversSetupBeforeUserCode(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.SearchTimeout = 40 * time.Millisecond

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(cfg, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	searchExec, ok := exec.(*searchExecutor)
	if !ok {
		t.Fatalf("unexpected executor type: %T", exec)
	}
	searchExec.beforeLoadGlobals = func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	_, err = exec.Execute(context.Background(), `return { ok: true };`, ModeSearch)
	if err == nil {
		t.Fatalf("expected error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T (%v)", err, err)
	}
}

func TestOutputTooLarge(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.MaxOutputBytes = 32

	artifacts := targetintersight.Artifacts()
	exec, err := NewSearchExecutor(cfg, artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog)
	if err != nil {
		t.Fatalf("NewSearchExecutor() error = %v", err)
	}
	defer exec.Close()

	_, err = exec.Execute(context.Background(), `return { data: "abcdefghijklmnopqrstuvwxyz" };`, ModeSearch)
	if err == nil {
		t.Fatalf("expected error")
	}

	var tooLarge contracts.OutputTooLarge
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected OutputTooLarge, got %T", err)
	}
}

func TestSpecOnlyExecutorRejectsQueryMode(t *testing.T) {
	t.Parallel()

	exec, err := NewQJSExecutorWithSpec(testConfig(), stubAPICaller{}, []byte(testSDKSpec))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithSpec() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `return await sdk.example.widget.list();`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Error() != "sdk runtime is not configured for query or mutate execution" {
		t.Fatalf("unexpected error: %v", validationErr)
	}
}

func TestAPICallLimit(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.MaxAPICalls = 1

	exec, err := newTestQJSExecutorWithArtifacts(cfg, stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	result, err := exec.Execute(context.Background(), `
await sdk.example.widget.get({ path: { Moid: 'widget-1' } });
try {
  await sdk.example.widget.list();
  return { ok: false };
} catch (err) {
  return err;
}
`, ModeQuery)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result.Value)
	}
	if payload["kind"] != "limit" {
		t.Fatalf("unexpected limit kind: %#v", payload["kind"])
	}
	if payload["message"] != "API call limit reached (1/1)" {
		t.Fatalf("unexpected limit message: %#v", payload["message"])
	}
	if payload["limit"] != int64(1) {
		t.Fatalf("unexpected limit payload: %#v", payload["limit"])
	}
}

func TestAPICallLimitUncaughtNormalizesToLimitError(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.MaxAPICalls = 1

	exec, err := newTestQJSExecutorWithArtifacts(cfg, stubAPICaller{
		do: func(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	}, []byte(testSDKSpec), []byte(testSDKCatalog), []byte(testSDKRules))
	if err != nil {
		t.Fatalf("NewQJSExecutorWithArtifacts() error = %v", err)
	}

	_, err = exec.Execute(context.Background(), `
await sdk.example.widget.get({ path: { Moid: 'widget-1' } });
return await sdk.example.widget.list();
`, ModeQuery)
	if err == nil {
		t.Fatalf("expected error")
	}

	var limitErr contracts.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected LimitError, got %T (%v)", err, err)
	}
}

func testConfig() Config {
	cfg := DefaultConfig()
	cfg.SearchTimeout = 20 * time.Second
	cfg.GlobalTimeout = 20 * time.Second
	cfg.PerCallTimeout = 500 * time.Millisecond
	return cfg
}

type stubAPICaller struct {
	do func(ctx context.Context, operation contracts.OperationDescriptor) (any, error)
}

func (s stubAPICaller) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if s.do != nil {
		return s.do(ctx, operation)
	}
	return map[string]any{"method": operation.Method, "path": operation.Path}, nil
}
