package server

import (
	"testing"

	targetintersight "github.com/mimaurer/intersight-mcp/implementations/intersight"
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
	    "/devices": {
	      "get": {
	        "operationId": "ListDevices"
	      }
	    }
	  },
	  "schemas": {
	    "inventory.Device": {
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
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{},"schemas":{"inventory.Device":{}}}`,
		},
		{
			name: "no operations",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/devices":{}},"schemas":{"inventory.Device":{}}}`,
		},
		{
			name: "no valid operations",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/devices":{"notAMethod":{}}},"schemas":{"inventory.Device":{}}}`,
		},
		{
			name: "missing schemas",
			data: `{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{},"schemas":{}}`,
		},
		{
			name: "missing metadata",
			data: `{"paths":{"/devices":{"get":{"operationId":"ListDevices"}}},"schemas":{"inventory.Device":{}}}`,
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

func TestValidateEmbeddedSpecAcceptsNonIntersightShapes(t *testing.T) {
	t.Parallel()

	tests := []string{
		`{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/v2/resources":{"get":{"operationId":"ListResources"}}},"schemas":{"service.Resource":{}}}`,
		`{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/not-api":{"get":{"operationId":"GetX"}}},"schemas":{"other.Schema":{}}}`,
		`{"metadata":{"published_version":"1","source_url":"x","sha256":"y","retrieval_date":"2026-04-08"},"paths":{"/devices":{"get":{"operationId":"ListDevices"}}},"schemas":{}}`,
	}
	for _, data := range tests {
		if err := ValidateEmbeddedSpec([]byte(data)); err != nil {
			t.Fatalf("expected generic embedded spec validation to succeed for %s: %v", data, err)
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
    "compute.rackUnits.list": {
      "sdkMethod": "compute.rackUnits.list",
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
    "compute.rackUnits": {
      "path": "/api/v1/compute/RackUnits",
      "operations": ["list"]
    }
  },
  "resourceNames": ["compute.rackUnits"],
  "paths": {
    "/api/v1/compute/RackUnits": ["compute.rackUnits"],
    "/api/v1/compute/rackunits": ["compute.rackUnits"],
    "/compute/RackUnits": ["compute.rackUnits"],
    "/compute/rackunits": ["compute.rackUnits"]
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
    "compute.rackUnits.list": {
      "sdkMethod": "compute.rackUnits.list",
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
    "compute.rackUnits": {
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
  "resourceNames": ["compute.rackUnits"],
  "paths": {
    "/api/v1/compute/RackUnits": ["compute.rackUnits"],
    "/api/v1/compute/rackunits": ["compute.rackUnits"],
    "/compute/RackUnits": ["compute.rackUnits"],
    "/compute/rackunits": ["compute.rackUnits"]
  }
}`)

	if err := ValidateEmbeddedArtifacts(spec, catalog, rules, searchCatalog); err == nil {
		t.Fatalf("expected mismatched catalog validation failure")
	}
}

func TestValidateCommittedEmbeddedArtifacts(t *testing.T) {
	artifacts := targetintersight.Artifacts()
	if err := ValidateEmbeddedArtifactsWithRuleTemplates(artifacts.ResolvedSpec, artifacts.SDKCatalog, artifacts.Rules, artifacts.SearchCatalog, targetintersight.RuleTemplates()); err != nil {
		t.Fatalf("ValidateEmbeddedArtifacts(committed generated artifacts) error = %v", err)
	}
}
