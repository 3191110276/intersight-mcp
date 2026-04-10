package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestGeneratorDefaultRouteRetentionAndNormalization(t *testing.T) {
	t.Parallel()

	out, catalog, rules, search, stdout, stderr := runFixtureGenerator(t, fixtureInputs{
		spec: baseFixtureSpec,
		filter: `
denylist:
  namespaces: []
  pathPrefixes: []
  operationIds: []
`,
	})

	if len(stdout) == 0 {
		t.Fatalf("expected machine-readable summary on stdout")
	}
	if len(stderr) == 0 {
		t.Fatalf("expected human-readable summary on stderr")
	}

	var spec normalizedSpec
	if err := json.Unmarshal(out, &spec); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if spec.Metadata.PublishedVersion != "1.0.0-fixture" {
		t.Fatalf("expected source metadata on spec, got %#v", spec.Metadata)
	}

	if _, ok := spec.Paths["/api/v1/compute/RackUnits"]; !ok {
		t.Fatalf("expected retained /api/v1 route")
	}
	if _, ok := spec.Paths["/other/path"]; ok {
		t.Fatalf("did not expect non-/api/v1 path to be retained")
	}

	get := spec.Paths["/api/v1/compute/RackUnits"]["get"]
	if get.OperationID != "GetComputeRackUnitList" {
		t.Fatalf("unexpected operationId: %q", get.OperationID)
	}
	if len(get.Parameters) != 2 {
		t.Fatalf("expected merged path+operation parameters, got %d", len(get.Parameters))
	}
	if get.Parameters[0].Name != "$top" || get.Parameters[1].Name != "$filter" {
		t.Fatalf("unexpected parameter merge result: %#v", get.Parameters)
	}

	post := spec.Paths["/api/v1/compute/RackUnits"]["post"]
	if post.RequestBody == nil || post.RequestBody.Content["application/json"].Schema == nil {
		t.Fatalf("expected normalized JSON request body")
	}
	if got := post.RequestBody.Content["application/json"].Schema.Required; len(got) != 1 || got[0] != "Name" {
		t.Fatalf("expected flattened request schema required fields, got %v", got)
	}

	rackUnit := spec.Schemas["compute.RackUnit"]
	if rackUnit.Properties["Organization"].ExpandTarget != "organization.Organization" {
		t.Fatalf("expected MoRef expand target, got %#v", rackUnit.Properties["Organization"])
	}
	if !rackUnit.Properties["Organization"].Relationship {
		t.Fatalf("expected relationship metadata, got %#v", rackUnit.Properties["Organization"])
	}
	if rackUnit.Properties["Organization"].RelationshipTarget != "organization.Organization" {
		t.Fatalf("expected relationship target, got %#v", rackUnit.Properties["Organization"])
	}
	if got := rackUnit.Properties["Organization"].RelationshipWriteForms; len(got) != 2 || got[0] != "moidRef" || got[1] != "typedMoRef" {
		t.Fatalf("expected relationship write forms, got %#v", got)
	}
	if got := rackUnit.Properties["Organization"].OneOf; len(got) != 2 {
		t.Fatalf("expected write-oriented relationship alternatives, got %#v", got)
	}
	if rackUnit.Properties["Children"].Items == nil || rackUnit.Properties["Children"].Items.Circular != "compute.RackUnit" {
		t.Fatalf("expected circular sentinel, got %#v", rackUnit.Properties["Children"].Items)
	}
	if rackUnit.Properties["Name"].Type != "string" {
		t.Fatalf("expected flattened inherited property, got %#v", rackUnit.Properties["Name"])
	}
	if got := rackUnit.Properties["AdminState"].Enum; len(got) != 2 || got[0] != "Enabled" || got[1] != "Disabled" {
		t.Fatalf("expected enum values to be preserved, got %#v", got)
	}
	if rackUnit.Properties["Organization"].Properties == nil {
		t.Fatalf("expected relationship schema to expose write-oriented properties, got %#v", rackUnit.Properties["Organization"])
	}

	if _, ok := spec.Schemas["unused.Unreachable"]; ok {
		t.Fatalf("did not expect unreachable schema to be retained")
	}

	var sdkCatalog contracts.SDKCatalog
	if err := json.Unmarshal(catalog, &sdkCatalog); err != nil {
		t.Fatalf("unmarshal sdk catalog: %v", err)
	}
	entry, ok := sdkCatalog.Methods["compute.rackUnit.list"]
	if !ok {
		t.Fatalf("expected compute.rackUnit.list in sdk catalog")
	}
	if entry.Descriptor.OperationID != "GetComputeRackUnitList" {
		t.Fatalf("unexpected descriptor operationId: %#v", entry.Descriptor)
	}
	if got := entry.QueryParameters; len(got) != 2 || got[0] != "$filter" || got[1] != "$top" {
		t.Fatalf("unexpected query parameters: %#v", got)
	}
	create, ok := sdkCatalog.Methods["compute.rackUnit.create"]
	if !ok {
		t.Fatalf("expected compute.rackUnit.create in sdk catalog")
	}
	if !create.RequestBodyRequired {
		t.Fatalf("expected create operation request body to be required")
	}
	if got := create.RequestBodyFields; len(got) != 4 || got[0] != "AdminState" || got[1] != "Children" || got[2] != "Name" || got[3] != "Organization" {
		t.Fatalf("unexpected request body fields: %#v", got)
	}
	if entry.Resource != "compute.RackUnit" {
		t.Fatalf("list resource = %q, want compute.RackUnit", entry.Resource)
	}
	if create.Resource != "compute.RackUnit" {
		t.Fatalf("create resource = %q, want compute.RackUnit", create.Resource)
	}

	var ruleCatalog contracts.RuleCatalog
	if err := json.Unmarshal(rules, &ruleCatalog); err != nil {
		t.Fatalf("unmarshal rules: %v", err)
	}
	if ruleCatalog.Metadata.PublishedVersion != "1.0.0-fixture" {
		t.Fatalf("expected source metadata on rules, got %#v", ruleCatalog.Metadata)
	}

	var searchCatalog contracts.SearchCatalog
	if err := json.Unmarshal(search, &searchCatalog); err != nil {
		t.Fatalf("unmarshal search catalog: %v", err)
	}
	resource, ok := searchCatalog.Resources["compute.rackUnit"]
	if !ok {
		t.Fatalf("expected compute.rackUnit in search catalog")
	}
	if got := resource.Schema; got != "compute.RackUnit" {
		t.Fatalf("search list resource = %q, want compute.RackUnit", got)
	}
	if got := resource.CreateFields["Name"].Type; got != "string" {
		t.Fatalf("search fields Name.type = %q, want string", got)
	}
	if got := resource.CreateFields["Children"].Items; got != "compute.RackUnit" {
		t.Fatalf("search fields Children.items = %q, want compute.RackUnit", got)
	}
	if got := resource.CreateFields["Organization"].Ref; got != "organization.Organization" {
		t.Fatalf("search fields Organization.ref = %q, want organization.Organization", got)
	}
	if !resource.CreateFields["AdminState"].Enum {
		t.Fatalf("expected search fields AdminState.enum to be true")
	}
	if !slices.Contains(resource.Operations, "create") {
		t.Fatalf("expected compute.rackUnit.create in search catalog resource: %#v", resource.Operations)
	}
}

func TestGeneratorExplicitDenylistPruning(t *testing.T) {
	t.Parallel()

	out, _, _, _, _, _ := runFixtureGenerator(t, fixtureInputs{
		spec: baseFixtureSpec,
		filter: `
denylist:
  namespaces: []
  pathPrefixes: []
  operationIds:
    - id: CreateComputeRackUnit
      rationale: remove writes from fixture
`,
	})

	var spec normalizedSpec
	if err := json.Unmarshal(out, &spec); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	methods := spec.Paths["/api/v1/compute/RackUnits"]
	if _, ok := methods["post"]; ok {
		t.Fatalf("expected post operation to be pruned")
	}
	if _, ok := methods["get"]; !ok {
		t.Fatalf("expected get operation to remain")
	}
}

func TestGeneratorManifestMismatchFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "third_party", "intersight", "openapi", "raw", "openapi.json")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(specPath, []byte(baseFixtureSpec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	manifestPath := filepath.Join(dir, "third_party", "intersight", "openapi", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"published_version":"wrong","source_url":"https://example.com","sha256":"deadbeef","retrieval_date":"2026-04-07"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	filterPath := filepath.Join(dir, "spec", "filter.yaml")
	if err := os.MkdirAll(filepath.Dir(filterPath), 0o755); err != nil {
		t.Fatalf("mkdir filter dir: %v", err)
	}
	if err := os.WriteFile(filterPath, []byte("denylist:\n  namespaces: []\n  pathPrefixes: []\n  operationIds: []\n"), 0o644); err != nil {
		t.Fatalf("write filter: %v", err)
	}

	outPath := filepath.Join(dir, "generated", "spec_resolved.json")
	err := newGenerator(specPath, filterPath, outPath, &bytes.Buffer{}, &bytes.Buffer{}).Run()
	if err == nil {
		t.Fatalf("expected manifest mismatch error")
	}
}

func TestGeneratorAcceptsYAMLRawSpecForManifestValidation(t *testing.T) {
	t.Parallel()

	out, _, _, _, _, _ := runFixtureGenerator(t, fixtureInputs{
		spec: fixtureYAMLSpec,
		filter: `
denylist:
  namespaces: []
  pathPrefixes: []
  operationIds: []
`,
	})

	var spec normalizedSpec
	if err := json.Unmarshal(out, &spec); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, ok := spec.Paths["/api/v1/example/Parents"]; !ok {
		t.Fatalf("expected YAML fixture route to be retained")
	}
}

func TestGeneratorExpandsDirectRefWriteBodiesForSearchCatalog(t *testing.T) {
	t.Parallel()

	_, _, _, search, _, _ := runFixtureGenerator(t, fixtureInputs{
		spec: `{
  "openapi": "3.0.2",
  "info": {
    "title": "Fixture",
    "version": "1.0.0-fixture"
  },
  "paths": {
    "/api/v1/ntp/Policies": {
      "post": {
        "summary": "Create a 'ntp.Policy' resource.",
        "operationId": "CreateNtpPolicy",
        "tags": ["ntp"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/ntp.Policy"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ntp.Policy"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "mo.MoRef": {
        "type": "object",
        "properties": {
          "Moid": { "type": "string" },
          "ObjectType": { "type": "string" }
        },
        "required": ["Moid", "ObjectType"]
      },
      "organization.Organization": {
        "type": "object",
        "properties": {
          "Name": { "type": "string" }
        }
      },
      "organization.OrganizationRelationship": {
        "allOf": [
          { "$ref": "#/components/schemas/mo.MoRef" },
          { "$ref": "#/components/schemas/organization.Organization" }
        ]
      },
      "ntp.Policy": {
        "type": "object",
        "required": ["ClassId", "ObjectType", "Enabled", "Timezone"],
        "properties": {
          "ClassId": { "type": "string", "enum": ["ntp.Policy"] },
          "ObjectType": { "type": "string", "enum": ["ntp.Policy"] },
          "Enabled": { "type": "boolean" },
          "Timezone": { "type": "string", "enum": ["UTC"] },
          "NtpServers": {
            "type": "array",
            "items": { "type": "string" }
          },
          "AuthenticatedNtpServers": {
            "type": "array",
            "items": { "type": "string" }
          },
          "Organization": {
            "$ref": "#/components/schemas/organization.OrganizationRelationship"
          }
        }
      }
    }
  }
}`,
		filter: `
denylist:
  namespaces: []
  pathPrefixes: []
  operationIds: []
`,
	})

	var searchCatalog contracts.SearchCatalog
	if err := json.Unmarshal(search, &searchCatalog); err != nil {
		t.Fatalf("unmarshal search catalog: %v", err)
	}
	if !slices.Contains(searchCatalog.Resources["ntp.policy"].Operations, "create") {
		t.Fatalf("expected ntp.policy.create in search catalog")
	}
}

type fixtureInputs struct {
	spec   string
	filter string
}

func runFixtureGenerator(t *testing.T, in fixtureInputs) ([]byte, []byte, []byte, []byte, []byte, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "third_party", "intersight", "openapi", "raw", "openapi.json")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatalf("mkdir spec dir: %v", err)
	}
	if err := os.WriteFile(specPath, []byte(in.spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	sum := sha256.Sum256([]byte(in.spec))
	manifest := `{"published_version":"1.0.0-fixture","source_url":"https://example.com/spec","sha256":"` + hex.EncodeToString(sum[:]) + `","retrieval_date":"2026-04-07"}`
	manifestPath := filepath.Join(dir, "third_party", "intersight", "openapi", "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	filterPath := filepath.Join(dir, "spec", "filter.yaml")
	if err := os.MkdirAll(filepath.Dir(filterPath), 0o755); err != nil {
		t.Fatalf("mkdir filter dir: %v", err)
	}
	if err := os.WriteFile(filterPath, []byte(in.filter), 0o644); err != nil {
		t.Fatalf("write filter: %v", err)
	}

	outPath := filepath.Join(dir, "generated", "spec_resolved.json")
	var stdout, stderr bytes.Buffer
	if err := newGenerator(specPath, filterPath, outPath, &stdout, &stderr).Run(); err != nil {
		t.Fatalf("run generator: %v", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	catalogPath := filepath.Join(dir, "generated", "sdk_catalog.json")
	catalog, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read sdk catalog: %v", err)
	}
	rulesPath := filepath.Join(dir, "generated", "rules.json")
	rules, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	searchPath := filepath.Join(dir, "generated", "search_catalog.json")
	search, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("read search catalog: %v", err)
	}
	return out, catalog, rules, search, stdout.Bytes(), stderr.Bytes()
}

const baseFixtureSpec = `{
  "openapi": "3.0.2",
  "info": {
    "title": "Fixture",
    "version": "1.0.0-fixture"
  },
  "tags": [
    {"name": "compute", "description": "Compute resources"},
    {"name": "unused"}
  ],
  "paths": {
    "/api/v1/compute/RackUnits": {
      "parameters": [
        {
          "name": "$top",
          "in": "query",
          "required": false,
          "schema": { "type": "integer", "format": "int32" }
        }
      ],
      "get": {
        "summary": "List rack units",
        "operationId": "GetComputeRackUnitList",
        "tags": ["compute"],
        "parameters": [
          {
            "name": "$filter",
            "in": "query",
            "required": false,
            "schema": { "type": "string" }
          }
        ],
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "Results": {
                      "type": "array",
                      "items": {
                        "$ref": "#/components/schemas/compute.RackUnit"
                      }
                    }
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "summary": "Create rack unit",
        "operationId": "CreateComputeRackUnit",
        "tags": ["compute"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "allOf": [
                  { "$ref": "#/components/schemas/compute.RackUnit" },
                  {
                    "type": "object",
                    "required": ["Name"],
                    "properties": {
                      "Name": { "type": "string" }
                    }
                  }
                ]
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "created",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/compute.RackUnit" }
              }
            }
          }
        }
      }
    },
    "/other/path": {
      "get": {
        "operationId": "Ignored",
        "responses": {
          "200": { "description": "ignored" }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "mo.BaseMo": {
        "type": "object",
        "properties": {
          "Name": { "type": "string", "description": "drop me" }
        },
        "required": ["Name"]
      },
      "mo.MoRef": {
        "type": "object",
        "properties": {
          "Moid": { "type": "string" },
          "ObjectType": { "type": "string" }
        },
        "required": ["Moid", "ObjectType"]
      },
      "organization.Organization": {
        "type": "object",
        "properties": {
          "Name": { "type": "string" }
        }
      },
      "organization.OrganizationRelationship": {
        "allOf": [
          { "$ref": "#/components/schemas/mo.MoRef" },
          { "$ref": "#/components/schemas/organization.Organization" }
        ]
      },
      "compute.RackUnit": {
        "allOf": [
          { "$ref": "#/components/schemas/mo.BaseMo" },
          {
            "type": "object",
            "properties": {
              "Organization": {
                "$ref": "#/components/schemas/organization.OrganizationRelationship"
              },
              "AdminState": {
                "type": "string",
                "enum": ["Enabled", "Disabled"]
              },
              "Children": {
                "type": "array",
                "items": { "$ref": "#/components/schemas/compute.RackUnit" }
              }
            }
          }
        ]
      },
      "unused.Unreachable": {
        "type": "object",
        "properties": {
          "Ghost": { "type": "string" }
        }
      }
    }
  }
}`

const fixtureYAMLSpec = `
openapi: 3.0.2
info:
  title: Fixture
  version: 1.0.0-fixture
paths:
  /api/v1/example/Parents:
    get:
      operationId: GetParentList
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/example.Parent'
components:
  schemas:
    example.Parent:
      type: object
      properties:
        Name:
          type: string
`
