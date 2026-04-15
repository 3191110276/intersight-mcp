package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	targetintersight "github.com/mimaurer/intersight-mcp/implementations/intersight"
	"github.com/mimaurer/intersight-mcp/internal/bootstrap"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

type testTarget struct {
	artifacts  implementations.Artifacts
	connection implementations.ConnectionConfig
}

func (t testTarget) Name() string { return "test" }

func (t testTarget) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Test Provider",
		ServerName:      "test-mcp",
		ConfigPrefix:    "INTERSIGHT",
		DefaultEndpoint: "https://intersight.com",
		AuthErrorHint:   "Check INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET.",
	}
}

func (t testTarget) Artifacts() implementations.Artifacts { return t.artifacts }

func (t testTarget) GenerationConfig() implementations.GenerationConfig {
	return implementations.GenerationConfig{}
}

func (t testTarget) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (t testTarget) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	if t.connection != nil {
		return t.connection, nil
	}
	return targetintersight.LoadConnectionConfig(args, environ)
}

func validTarget() implementations.Target {
	return testTarget{artifacts: validArtifacts()}
}

func testApp(target implementations.Target) bootstrap.App {
	return bootstrap.App{
		Target:  target,
		Version: "test",
	}
}

func validArtifacts() implementations.Artifacts {
	return implementations.Artifacts{
		ResolvedSpec:  validTestSpec,
		SDKCatalog:    validTestCatalog,
		Rules:         validTestRules,
		SearchCatalog: validTestSearchCatalog,
	}
}

func TestServeStartsWithoutCredentialsForOfflineSearch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := testApp(validTarget()).ServeWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, nil); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
}

func TestServeReadOnlyOmitsMutateTool(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stdinReader, stdinWriter := io.Pipe()
	var stdout bytes.Buffer

	go func() {
		writeJSONLine(t, stdinWriter, initializeRequest())
		writeJSONLine(t, stdinWriter, toolsListRequest(2))
		_ = stdinWriter.Close()
	}()

	if err := testApp(validTarget()).ServeWithIO(ctx, []string{"--read-only"}, stdinReader, &stdout, &bytes.Buffer{}, nil); err != nil {
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

func TestServeWarnsWhenUnsafeCodeLoggingEnabled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	err := testApp(validTarget()).ServeWithIO(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &stderr, []string{
		"INTERSIGHT_LOG_LEVEL=debug",
		"INTERSIGHT_UNSAFE_LOG_FULL_CODE=true",
	})
	if err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "unsafe full-code debug logging is enabled") {
		t.Fatalf("expected unsafe logging warning, got: %s", stderr.String())
	}
}

func TestNewHTTPClientDisablesAmbientProxyByDefault(t *testing.T) {
	t.Parallel()

	client, err := bootstrap.NewHTTPClient(time.Second, "")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type: %T", client.Transport)
	}
	if transport.Proxy == nil {
		return
	}

	req := &http.Request{URL: mustParseURL(t, "https://intersight.com/api/v1/compute/RackUnits")}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() error = %v", err)
	}
	if proxyURL != nil {
		t.Fatalf("expected no proxy, got %q", proxyURL)
	}
}

func TestNewHTTPClientUsesExplicitProxy(t *testing.T) {
	t.Parallel()

	client, err := bootstrap.NewHTTPClient(time.Second, "http://proxy.example.com:8080")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type: %T", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("expected explicit proxy function")
	}

	req := &http.Request{URL: mustParseURL(t, "https://intersight.com/api/v1/compute/RackUnits")}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("transport.Proxy() error = %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://proxy.example.com:8080" {
		t.Fatalf("unexpected proxy URL: %#v", proxyURL)
	}
}

func TestNewHTTPClientFailsOnInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := bootstrap.NewHTTPClient(time.Second, "://bad-proxy")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `configure HTTP client proxy: invalid proxy URL "://bad-proxy"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServeFailsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := testApp(validTarget()).ServeWithIO(context.Background(), []string{"--endpoint", "not-a-url/path"}, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env)
	if err == nil || !strings.Contains(err.Error(), "invalid endpoint") {
		t.Fatalf("ServeWithIO() error = %v, want invalid endpoint failure", err)
	}
}

func TestServeFailsOnInvalidProxyConfig(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := testApp(validTarget()).ServeWithIO(context.Background(), []string{"--proxy", "://bad-proxy"}, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env)
	if err == nil || !strings.Contains(err.Error(), "invalid proxy") {
		t.Fatalf("ServeWithIO() error = %v, want invalid proxy failure", err)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return parsed
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServeFailsOnMalformedEmbeddedArtifacts(t *testing.T) {
	t.Parallel()

	env := []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	}
	err := testApp(testTarget{artifacts: implementations.Artifacts{
		ResolvedSpec:  []byte(`{`),
		SDKCatalog:    []byte(`{}`),
		Rules:         []byte(`{}`),
		SearchCatalog: []byte(`{}`),
	}}).ServeWithIO(context.Background(), nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env)
	if err == nil || !strings.Contains(err.Error(), "embedded") {
		t.Fatalf("ServeWithIO() error = %v, want embedded artifact failure", err)
	}
}

func TestServeStartsWhenAuthBootstrapFails(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4TLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if err := testApp(validTarget()).ServeWithIOAndHTTPClient(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, api.Client()); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
}

func TestServeStartsWithoutBlockingOnAuthBootstrap(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4TLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			<-r.Context().Done()
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

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	start := time.Now()
	if err := testApp(validTarget()).ServeWithIOAndHTTPClient(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, api.Client()); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("ServeWithIO() took %v, want startup to avoid blocking on auth bootstrap", elapsed)
	}
}

func TestServeRetriesAuthBootstrapAfterStartupFailure(t *testing.T) {
	t.Parallel()

	var allowAuth atomic.Bool
	api := testutil.NewTCP4TLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			if !allowAuth.Load() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"bad credentials"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		case "/api/v1/iam/UserPreferences":
			if !allowAuth.Load() {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Results":[]}`))
		case "/api/v1/compute/RackUnits":
			if auth := r.Header.Get("Authorization"); auth != "Bearer token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Results":[{"Moid":"rack-1"}]}`))
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

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()

	errCh := make(chan error, 1)
	lineCh := make(chan string, 4)

	go func() {
		errCh <- testApp(validTarget()).ServeWithIOAndHTTPClient(context.Background(), nil, stdinReader, stdoutWriter, &bytes.Buffer{}, env, api.Client())
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
	allowAuth.Store(true)
	writeJSONLine(t, stdinWriter, toolCallRequest(2, "query", `return await sdk.compute.rackUnits.list();`))

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
			t.Fatalf("ServeWithIO() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ServeWithIO to return")
	}

	responses := indexResponsesByID(t, lines)
	query := decodeToolResult(t, responses[2])
	if query.IsError {
		t.Fatalf("query IsError = true, want false")
	}
	envelope, ok := query.StructuredContent.(contracts.SuccessEnvelope)
	if !ok {
		t.Fatalf("unexpected query envelope type: %T", query.StructuredContent)
	}
	result, ok := envelope.Result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected query result type: %T", envelope.Result)
	}
	results, ok := result["Results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("unexpected query result payload: %#v", envelope.Result)
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
		errCh <- testApp(validTarget()).ServeWithIO(context.Background(), nil, stdinReader, stdoutWriter, &bytes.Buffer{}, nil)
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
	writeJSONLine(t, stdinWriter, toolCallRequest(2, "query", `return await sdk.compute.rackUnits.list();`))

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
			t.Fatalf("ServeWithIO() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ServeWithIO to return")
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
    "compute.rackUnits.list": {
      "sdkMethod": "compute.rackUnits.list",
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
    "compute.rackUnits": {
      "schema": "compute.RackUnit",
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

func TestServeWithIOGracefulOnClosedInput(t *testing.T) {
	t.Parallel()

	api := testutil.NewTCP4TLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if err := testApp(validTarget()).ServeWithIOAndHTTPClient(ctx, nil, bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{}, env, api.Client()); err != nil {
		t.Fatalf("ServeWithIO() error = %v", err)
	}
}
