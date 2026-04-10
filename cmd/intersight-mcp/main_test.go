package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
	"github.com/mimaurer/intersight-mcp/intersight"
)

func TestServeStartsWithoutCredentialsForOfflineSearch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := serveWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, nil, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog); err != nil {
		t.Fatalf("serveWithIO() error = %v", err)
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

func TestServeFailsOnMalformedEmbeddedArtifacts(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := serveWithIO(context.Background(), nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, []byte(`{`), []byte(`{}`), []byte(`{}`), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("serveWithIO() error = %v, want embedded artifact failure", err)
	}
}

func TestServeStartsWhenAuthBootstrapFails(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := serveWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog); err != nil {
		t.Fatalf("serveWithIO() error = %v", err)
	}
}

func TestBootstrapOAuthManagerTimesOutStalledStartupAuth(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	start := time.Now()
	_, err := bootstrapOAuthManager(context.Background(), 50*time.Millisecond, intersight.OAuthConfig{
		TokenURL:     api.URL + "/iam/token",
		ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   api.Client(),
	})
	if err == nil {
		t.Fatalf("expected startup auth timeout")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("bootstrapOAuthManager() took %v, want bounded timeout", elapsed)
	}
}

func TestBootstrapOAuthManagerKeepsProactiveRefreshAlive(t *testing.T) {
	t.Parallel()

	clock := testutil.NewManualClock(time.Unix(0, 0))
	var tokenCalls atomic.Int32

	api := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			token := "bootstrap-token"
			if tokenCalls.Add(1) > 1 {
				token = "refreshed-token"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"` + token + `","expires_in":8}`))
		case "/api/v1/iam/UserPreferences":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Results":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := bootstrapOAuthManager(ctx, time.Second, intersight.OAuthConfig{
		TokenURL:     api.URL + "/iam/token",
		ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   api.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("bootstrapOAuthManager() error = %v", err)
	}

	clock.Advance(4 * time.Second)

	deadline := time.Now().Add(time.Second)
	for tokenCalls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := tokenCalls.Load(); got < 2 {
		t.Fatalf("expected proactive refresh after bootstrap, got %d token requests", got)
	}

	token, err := manager.Token(ctx)
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "refreshed-token" {
		t.Fatalf("unexpected token after proactive refresh: %q", token)
	}
}

func TestServeWithoutCredentialsQueryReturnsAuthError(t *testing.T) {
	t.Parallel()

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	errCh := make(chan error, 1)
	lineCh := make(chan string, 4)

	go func() {
		errCh <- serveWithIO(context.Background(), nil, stdinReader, stdoutWriter, &bytes.Buffer{}, nil, validTestSpec, validTestCatalog, validTestRules, validTestSearchCatalog)
		_ = stdoutWriter.Close()
	}()
	go func() {
		scanner := bufio.NewScanner(stdoutReader)
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		close(lineCh)
	}()

	writeJSONLine(t, stdinWriter, initializeRequest())
	writeJSONLine(t, stdinWriter, toolCallRequest(2, "query", `return await sdk.compute.rackUnit.list();`))

	lines := make([]string, 0, 2)
	for len(lines) < 2 {
		select {
		case line, ok := <-lineCh:
			if !ok {
				t.Fatalf("stdout closed after %d responses, want 2", len(lines))
			}
			lines = append(lines, line)
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for MCP responses")
		}
	}
	_ = stdinWriter.Close()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serveWithIO() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for serveWithIO to return")
	}

	responses := indexResponsesByID(t, lines)
	query := decodeToolResult(t, responses[2])
	if !query.IsError {
		t.Fatalf("query IsError = false, want true")
	}
	envelope, ok := query.StructuredContent.(contracts.ErrorEnvelope)
	if !ok {
		t.Fatalf("unexpected query envelope type: %T", query.StructuredContent)
	}
	if envelope.Error.Type != contracts.ErrorTypeAuth {
		t.Fatalf("error.type = %q, want %q", envelope.Error.Type, contracts.ErrorTypeAuth)
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
