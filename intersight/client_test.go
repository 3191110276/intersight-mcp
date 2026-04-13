package intersight

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

func TestClientDoJSONSuccessAndEndpointOverride(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if got := r.URL.Query().Get("$top"); got != "5" {
			t.Fatalf("unexpected query value: %q", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"path":   r.URL.Path,
			"method": r.Method,
		})
	}))
	defer server.Close()

	client := NewClient(server.Client(), "https://unused.example.com", staticTokenProvider("test-token"))
	got, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/compute/RackUnits", RequestOptions{
		Query:       map[string]string{"$top": "5"},
		EndpointURL: server.URL + "/api/v1/compute/RackUnits",
	})
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}

	payload, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", got)
	}
	if payload["path"] != "/api/v1/compute/RackUnits" {
		t.Fatalf("unexpected path: %#v", payload["path"])
	}
	if payload["method"] != http.MethodGet {
		t.Fatalf("unexpected method: %#v", payload["method"])
	}
}

func TestClientDoJSONAcceptsAbsoluteAPIV1Paths(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/compute/RackUnits" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"path": r.URL.Path,
		})
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/api/v1", staticTokenProvider("test-token"))
	got, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/compute/RackUnits", RequestOptions{})
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}

	payload, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", got)
	}
	if payload["path"] != "/api/v1/compute/RackUnits" {
		t.Fatalf("unexpected payload path: %#v", payload["path"])
	}
}

func TestClientDoUsesPathTemplateAndRepeatedQueryAndHeaders(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/example/Widgets/widget/1" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query()["$select"]; len(got) != 2 || got[0] != "Name" || got[1] != "Moid" {
			t.Fatalf("unexpected repeated query values: %#v", got)
		}
		if got := r.Header.Values("X-Test"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("unexpected repeated header values: %#v", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"ok": true})
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL, staticTokenProvider("test-token"))
	_, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Kind:         contracts.OperationKindHTTP,
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/example/Widgets/{id}",
		PathParams:   map[string]string{"id": "widget/1"},
		QueryParams:  map[string][]string{"$select": []string{"Name", "Moid"}},
		Headers:      map[string][]string{"X-Test": []string{"a", "b"}},
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

func TestClientDoRejectsMissingPathTemplateParams(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{}, "https://example.com", staticTokenProvider("test-token"))
	_, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Kind:         contracts.OperationKindHTTP,
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/example/Widgets/{id}",
		PathParams:   map[string]string{},
		ResponseMode: contracts.ResponseModeJSON,
		ValidationPlan: contracts.ValidationPlan{
			Kind: contracts.ValidationPlanNone,
		},
		FollowUpPlan: contracts.FollowUpPlan{
			Kind: contracts.FollowUpPlanNone,
		},
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestClientDoJSONHTTPErrorNormalization(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusBadGateway, map[string]any{"message": "downstream error"})
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL, staticTokenProvider("test-token"))
	_, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/test", RequestOptions{})
	if err == nil {
		t.Fatalf("expected HTTP error")
	}

	var httpErr contracts.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.Status != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", httpErr.Status)
	}
}

func TestClientDoJSONTimeoutNormalization(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, &net.DNSError{IsTimeout: true}
		}),
	}, "https://example.com", staticTokenProvider("test-token"))

	_, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/test", RequestOptions{})
	if err == nil {
		t.Fatalf("expected timeout error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
}

func TestClientDoJSONNetworkNormalization(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		}),
	}, "https://example.com", staticTokenProvider("test-token"))

	_, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/test", RequestOptions{})
	if err == nil {
		t.Fatalf("expected network error")
	}

	var networkErr contracts.NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
}

func TestClientDoJSONRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"` + strings.Repeat("a", maxIntersightResponseBytes) + `"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL, staticTokenProvider("test-token"))
	_, err := client.DoJSON(context.Background(), http.MethodGet, "/api/v1/test", RequestOptions{})
	if err == nil {
		t.Fatalf("expected oversized response error")
	}

	var tooLarge contracts.OutputTooLarge
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected OutputTooLarge, got %T", err)
	}
	if tooLarge.Message != "Intersight response exceeded the 16 MiB limit" {
		t.Fatalf("unexpected error message: %q", tooLarge.Message)
	}
}

type staticTokenProvider string

func (s staticTokenProvider) Token(context.Context) (string, error) {
	return string(s), nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
