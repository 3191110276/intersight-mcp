package meraki

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientDoSuccessAndBearerAuth(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		if got := r.URL.Query().Get("perPage"); got != "5" {
			t.Fatalf("perPage = %q, want 5", got)
		}
		if got := r.Header.Get("X-Test"); got != "a" {
			t.Fatalf("X-Test = %q, want a", got)
		}
		if r.URL.Path != "/api/v1/networks" {
			t.Fatalf("path = %q, want /api/v1/networks", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	got, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Kind:         contracts.OperationKindHTTP,
		Method:       http.MethodGet,
		PathTemplate: "/networks",
		Path:         "/networks",
		QueryParams:  map[string][]string{"perPage": {"5"}},
		Headers:      map[string][]string{"X-Test": {"a"}},
		ResponseMode: contracts.ResponseModeJSON,
		ValidationPlan: contracts.ValidationPlan{
			Kind: contracts.ValidationPlanNone,
		},
		FollowUpPlan: contracts.FollowUpPlan{
			Kind: contracts.FollowUpPlanNone,
		},
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	payload, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", got)
	}
	if payload["ok"] != true {
		t.Fatalf("unexpected result payload: %#v", payload)
	}
}

func TestClientDoUsesEndpointOverride(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/networks" {
			t.Fatalf("path = %q, want /custom/networks", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), "https://unused.example.com/api/v1", "test-key")
	_, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Kind:         contracts.OperationKindHTTP,
		Method:       http.MethodGet,
		PathTemplate: "/networks",
		Path:         "/networks",
		EndpointURL:  server.URL + "/custom/networks",
		ResponseMode: contracts.ResponseModeJSON,
		ValidationPlan: contracts.ValidationPlan{
			Kind: contracts.ValidationPlanNone,
		},
		FollowUpPlan: contracts.FollowUpPlan{
			Kind: contracts.FollowUpPlanNone,
		},
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestClientDoMissingAPIKeyReturnsAuthError(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{}, "https://api.meraki.com/api/v1", "")
	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected auth error")
	}

	var authErr contracts.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T", err)
	}
}

func TestClientDoUnauthorizedReturnsAuthError(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":["bad key"]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "bad-key")
	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected auth error")
	}

	var authErr contracts.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T", err)
	}
}

func TestClientDoHTTPErrorNormalization(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"errors":["downstream"]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected HTTP error")
	}

	var httpErr contracts.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.Status != http.StatusBadGateway {
		t.Fatalf("Status = %d, want %d", httpErr.Status, http.StatusBadGateway)
	}
}

func TestClientDoTimeoutNormalization(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, &net.DNSError{IsTimeout: true}
		}),
	}, "https://api.meraki.com/api/v1", "test-key")

	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected timeout error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
}

func TestClientDoNetworkNormalization(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		}),
	}, "https://api.meraki.com/api/v1", "test-key")

	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected network error")
	}

	var networkErr contracts.NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
}

func TestClientDoRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"` + strings.Repeat("a", maxMerakiResponseBytes) + `"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatalf("expected size error")
	}

	var sizeErr contracts.OutputTooLarge
	if !errors.As(err, &sizeErr) {
		t.Fatalf("expected OutputTooLarge, got %T", err)
	}
}

func TestClientDoRetriesRateLimitTwiceThenSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors":["slow down"]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	got, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	payload, ok := got.(map[string]any)
	if !ok || payload["ok"] != true {
		t.Fatalf("unexpected result payload: %#v", got)
	}
}

func TestClientDoRateLimitReturnsRetryableHTTPErrorAfterRetries(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":["slow down"]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	_, err := client.Do(context.Background(), contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks"))
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	var httpErr contracts.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.Status != http.StatusTooManyRequests {
		t.Fatalf("Status = %d, want %d", httpErr.Status, http.StatusTooManyRequests)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestClientDoListAllFollowsPaginationLinks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("startingAfter") {
		case "":
			w.Header().Set("Link", "</api/v1/networks?startingAfter=cursor-1>; rel=\"next\"")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"n1"}]`))
		case "cursor-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"n2"}]`))
		default:
			t.Fatalf("unexpected startingAfter = %q", r.URL.Query().Get("startingAfter"))
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", "test-key")
	op := contracts.NewHTTPOperationDescriptor(http.MethodGet, "/networks")
	op.FollowUpPlan = contracts.FollowUpPlan{Kind: merakiListAllFollowUpKind}
	got, err := client.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	items, ok := got.([]any)
	if !ok {
		t.Fatalf("result type = %T, want []any", got)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestMerakiRetryDelayDefaultsWithoutRetryAfter(t *testing.T) {
	t.Parallel()

	got := merakiRetryDelay(http.Header{}, 2)
	if got != time.Second {
		t.Fatalf("delay = %v, want %v", got, time.Second)
	}
}
