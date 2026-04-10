package main

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

func TestServeFailsOnMissingCredentials(t *testing.T) {
	t.Parallel()

	err := serveWithIO(context.Background(), nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, nil, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog)
	if err == nil || !strings.Contains(err.Error(), "INTERSIGHT_CLIENT_ID") {
		t.Fatalf("serveWithIO() error = %v, want missing credentials", err)
	}
}

func TestServeFailsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := serveWithIO(context.Background(), []string{"--endpoint", "not-a-url"}, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog)
	if err == nil || !strings.Contains(err.Error(), "invalid endpoint") {
		t.Fatalf("serveWithIO() error = %v, want invalid endpoint failure", err)
	}
}

func TestServeFailsOnInvalidEmbeddedSpec(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := serveWithIO(context.Background(), nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, []byte(`{}`), []byte(`{}`), []byte(`{}`), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("serveWithIO() error = %v, want embedded spec failure", err)
	}
}

func TestServeFailsOnAuthBootstrapFailure(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"bad credentials"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=" + api.URL,
	}
	err := serveWithIO(context.Background(), nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog)
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("serveWithIO() error = %v, want auth bootstrap failure", err)
	}
}

var validTestSpec = []byte(`{
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

var validTestCatalog = []byte(`{
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

var validTestRules = []byte(`{
  "metadata": {
    "published_version": "1.0.0-test",
    "source_url": "https://example.com/spec",
    "sha256": "abc123",
    "retrieval_date": "2026-04-08"
  },
  "methods": {}
}`)

var validTestSearchCatalog = []byte(`{
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

func TestServeWithIOGracefulOnClosedInput(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		case "/api/v1/iam/UserPreferences":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Results":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=" + api.URL,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := serveWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog); err != nil {
		t.Fatalf("serveWithIO() error = %v", err)
	}
}
