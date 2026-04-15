package contracts

import (
	"strings"
	"testing"
)

func TestValidateSDKCatalogAgainstSpecRejectsUnknownRequestBodyField(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/compute/RackUnits": {
				"post": {
					OperationID: "CreateComputeRackUnit",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"Name": {},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit": {Type: "object"},
		},
	}

	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}
	method := catalog.Methods["compute.rackUnits.create"]
	method.RequestBodyFields = append(method.RequestBodyFields, "Bogus")
	catalog.Methods["compute.rackUnits.create"] = method

	err = ValidateSDKCatalogAgainstSpec(spec, catalog)
	if err == nil || !strings.Contains(err.Error(), "unknown request body field") {
		t.Fatalf("ValidateSDKCatalogAgainstSpec() error = %v, want unknown request body field failure", err)
	}
}

func TestValidateSDKCatalogAgainstSpecRejectsUnknownSchemaReference(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/compute/RackUnits": {
				"get": {
					OperationID: "GetComputeRackUnitList",
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit": {Type: "object"},
		},
	}

	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}
	method := catalog.Methods["compute.rackUnits.list"]
	method.RelatedSchemas = []string{"missing.Schema"}
	catalog.Methods["compute.rackUnits.list"] = method

	err = ValidateSDKCatalogAgainstSpec(spec, catalog)
	if err == nil || !strings.Contains(err.Error(), "unknown schema") {
		t.Fatalf("ValidateSDKCatalogAgainstSpec() error = %v, want unknown schema failure", err)
	}
}

func TestBuildSDKCatalogDerivesCanonicalResourceFromListEnvelopeAndSummaryFallback(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/compute/RackUnits": {
				"get": {
					OperationID: "GetComputeRackUnitList",
					Summary:     "Read a 'compute.RackUnit' resource.",
					Responses: map[string]NormalizedResponse{
						"200": {
							Content: map[string]NormalizedMediaContent{
								"application/json": {
									Schema: &NormalizedSchema{
										Circular: "compute.RackUnit.Response",
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/compute/RackUnits/{Moid}": {
				"delete": {
					OperationID: "DeleteComputeRackUnit",
					Summary:     "Delete a 'compute.RackUnit' resource.",
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit": {Type: "object"},
			"compute.RackUnit.Response": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Results": {
						Type:  "array",
						Items: &NormalizedSchema{Circular: "compute.RackUnit"},
					},
				},
			},
		},
	}

	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}

	if got := catalog.Methods["compute.rackUnits.list"].Resource; got != "compute.RackUnit" {
		t.Fatalf("list resource = %q, want compute.RackUnit", got)
	}
	if got := catalog.Methods["compute.rackUnits.delete"].Resource; got != "compute.RackUnit" {
		t.Fatalf("delete resource = %q, want compute.RackUnit", got)
	}
}

func TestBuildSDKCatalogDerivesMethodIDsFromNonAPIV1Paths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "versioned without api prefix",
			path: "/v2/compute/RackUnits",
			want: "compute.rackUnits.list",
		},
		{
			name: "unprefixed collection path",
			path: "/compute/RackUnits",
			want: "compute.rackUnits.list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := NormalizedSpec{
				Metadata: ArtifactSourceMetadata{
					PublishedVersion: "1.0.0-test",
					SourceURL:        "https://example.com/spec",
					SHA256:           "abc123",
					RetrievalDate:    "2026-04-08",
				},
				Paths: map[string]map[string]NormalizedOperation{
					tt.path: {
						"get": {
							OperationID: "ListComputeRackUnits",
						},
					},
				},
				Schemas: map[string]NormalizedSchema{
					"compute.RackUnit": {Type: "object"},
				},
			}

			catalog, err := BuildSDKCatalog(spec)
			if err != nil {
				t.Fatalf("BuildSDKCatalog() error = %v", err)
			}
			if _, ok := catalog.Methods[tt.want]; !ok {
				t.Fatalf("expected sdk method %q, got %#v", tt.want, catalog.Methods)
			}
		})
	}
}

func TestBuildSDKCatalogUsesPathResourceNameForInlineTopLevelItemPaths(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/devices/{serial}": {
				"get": {
					OperationID: "GetDevice",
					Parameters: []NormalizedParameter{
						{Name: "serial", In: "path", Required: true, Schema: &NormalizedSchema{Type: "string"}},
					},
					Responses: map[string]NormalizedResponse{
						"200": {
							Content: map[string]NormalizedMediaContent{
								"application/json": {
									Schema: &NormalizedSchema{Circular: "inline.GetDevice.response.200"},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"inline.GetDevice.response.200": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"serial": {Type: "string"},
				},
			},
		},
	}

	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}
	if _, ok := catalog.Methods["devices.devices.get"]; !ok {
		t.Fatalf("expected top-level item path to derive devices.devices.get, got %#v", catalog.Methods)
	}
}

func TestBuildSDKCatalogResolvesNestedGETNameCollisions(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-14",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/devices/{serial}/sensor/commands": {
				"get": {OperationID: "getDeviceSensorCommands"},
			},
			"/devices/{serial}/sensor/commands/{commandId}": {
				"get": {OperationID: "getDeviceSensorCommand"},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"device.SensorCommand": {Type: "object"},
		},
	}

	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}
	if _, ok := catalog.Methods["devices.sensorcommands.list"]; !ok {
		t.Fatalf("expected devices.sensorcommands.list, got %#v", catalog.Methods)
	}
	if _, ok := catalog.Methods["devices.sensorcommands.get"]; !ok {
		t.Fatalf("expected devices.sensorcommands.get, got %#v", catalog.Methods)
	}
}
