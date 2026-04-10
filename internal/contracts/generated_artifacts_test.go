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
	method := catalog.Methods["compute.rackUnit.create"]
	method.RequestBodyFields = append(method.RequestBodyFields, "Bogus")
	catalog.Methods["compute.rackUnit.create"] = method

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
	method := catalog.Methods["compute.rackUnit.list"]
	method.RelatedSchemas = []string{"missing.Schema"}
	catalog.Methods["compute.rackUnit.list"] = method

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

	if got := catalog.Methods["compute.rackUnit.list"].Resource; got != "compute.RackUnit" {
		t.Fatalf("list resource = %q, want compute.RackUnit", got)
	}
	if got := catalog.Methods["compute.rackUnit.delete"].Resource; got != "compute.RackUnit" {
		t.Fatalf("delete resource = %q, want compute.RackUnit", got)
	}
}
