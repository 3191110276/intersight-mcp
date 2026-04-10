package server

import (
	"testing"

	"github.com/mimaurer/intersight-mcp/generated"
)

func TestValidateEmbeddedSpecSuccess(t *testing.T) {
	t.Parallel()

	err := ValidateEmbeddedSpec([]byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "paths": {
    "/api/v1/compute/RackUnits": {
      "get": {
        "operationId": "GetComputeRackUnitList"
      }
    }
  },
  "schemas": {
    "compute.RackUnit": {
      "type": "object"
    }
  },
  "tags": []
}`))
	if err != nil {
		t.Fatalf("ValidateEmbeddedSpec() error = %v", err)
	}
}

func TestValidateEmbeddedSpecStructuralFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "empty paths",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{},"schemas":{"compute.RackUnit":{}}}`,
		},
		{
			name: "no operations",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/api/v1/x":{}},"schemas":{"compute.RackUnit":{}}}`,
		},
		{
			name: "no valid operations",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/api/v1/x":{"notAMethod":{}}},"schemas":{"compute.RackUnit":{}}}`,
		},
		{
			name: "missing schemas",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/api/v1/x":{"get":{"operationId":"GetX"}}},"schemas":{}}`,
		},
		{
			name: "missing metadata",
			data: `{"paths":{"/api/v1/x":{"get":{"operationId":"GetX"}}},"schemas":{"compute.RackUnit":{}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateEmbeddedSpec([]byte(tt.data)); err == nil {
				t.Fatalf("expected validation failure")
			}
		})
	}
}

func TestValidateEmbeddedSpecSemanticFailures(t *testing.T) {
	t.Parallel()

	tests := []string{
		`{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/not-api":{"get":{"operationId":"GetX"}}},"schemas":{"compute.RackUnit":{}}}`,
		`{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/api/v1/x":{"get":{"operationId":"GetX"}}},"schemas":{"other.Schema":{}}}`,
	}
	for _, data := range tests {
		if err := ValidateEmbeddedSpec([]byte(data)); err == nil {
			t.Fatalf("expected semantic validation failure for %s", data)
		}
	}
}

func TestValidateEmbeddedArtifacts(t *testing.T) {
	t.Parallel()

	spec := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "paths": {
    "/api/v1/compute/RackUnits": {
      "get": {
        "operationId": "GetComputeRackUnitList"
      }
    }
  },
  "schemas": {
    "compute.RackUnit": {
      "type": "object"
    }
  },
  "tags": []
}`)
	catalog := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {
    "compute.rackUnit.list": {
      "sdkMethod": "compute.rackUnit.list",
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
    }
  }
}`)
	rules := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {}
}`)
	searchCatalog := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "resources": {
    "compute.rackUnit": {
      "schema": "compute.RackUnit",
      "path": "/api/v1/compute/RackUnits",
      "operations": ["list"]
    }
  },
  "resourceNames": ["compute.rackUnit"],
  "paths": {
    "/api/v1/compute/RackUnits": ["compute.rackUnit"],
    "/api/v1/compute/rackunits": ["compute.rackUnit"],
    "/compute/RackUnits": ["compute.rackUnit"],
    "/compute/rackunits": ["compute.rackUnit"]
  }
}`)

	if err := ValidateEmbeddedArtifacts(spec, catalog, rules, searchCatalog); err != nil {
		t.Fatalf("ValidateEmbeddedArtifacts() error = %v", err)
	}
}

func TestValidateEmbeddedArtifactsRejectsMismatchedCatalog(t *testing.T) {
	t.Parallel()

	spec := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "paths": {
    "/api/v1/compute/RackUnits": {
      "get": {
        "operationId": "GetComputeRackUnitList"
      }
    }
  },
  "schemas": {
    "compute.RackUnit": {
      "type": "object"
    }
  },
  "tags": []
}`)
	catalog := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {
    "compute.rackUnit.list": {
      "sdkMethod": "compute.rackUnit.list",
      "resource": "compute.RackUnit",
      "descriptor": {
        "kind": "http-operation",
        "operationId": "WrongOperation",
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
    }
  }
}`)
	rules := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {}
}`)
	searchCatalog := []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "resources": {
    "compute.rackUnit": {
      "schema": "compute.RackUnit",
      "operations": {
        "list": {
          "kind": "read",
          "operationId": "WrongOperation",
          "method": "GET",
          "path": "/api/v1/compute/RackUnits"
        }
      }
    }
  },
  "resourceNames": ["compute.rackUnit"],
  "paths": {
    "/api/v1/compute/RackUnits": ["compute.rackUnit"],
    "/api/v1/compute/rackunits": ["compute.rackUnit"],
    "/compute/RackUnits": ["compute.rackUnit"],
    "/compute/rackunits": ["compute.rackUnit"]
  }
}`)

	if err := ValidateEmbeddedArtifacts(spec, catalog, rules, searchCatalog); err == nil {
		t.Fatalf("expected mismatched catalog validation failure")
	}
}

func TestValidateCommittedEmbeddedArtifacts(t *testing.T) {
	if err := ValidateEmbeddedArtifacts(generated.ResolvedSpecBytes(), generated.SDKCatalogBytes(), generated.RulesBytes(), generated.SearchCatalogBytes()); err != nil {
		t.Fatalf("ValidateEmbeddedArtifacts(committed generated artifacts) error = %v", err)
	}
}
