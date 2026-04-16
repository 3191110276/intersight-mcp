package nexusdashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestClientDoWithBearerToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		if got := r.Header.Get("Cookie"); got != "AuthCookie=test-token" {
			t.Fatalf("Cookie = %q, want AuthCookie=test-token", got)
		}
		if r.URL.Path != "/api/v1/manage/fabrics" {
			t.Fatalf("path = %q, want /api/v1/manage/fabrics", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[{"name":"fabric-a"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), ConnectionConfig{
		Endpoint: server.URL,
		Token:    "test-token",
	})

	got, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/manage/fabrics",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	result, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", got)
	}
	if _, ok := result["items"]; !ok {
		t.Fatalf("result = %#v, want items key", result)
	}
}

func TestClientDoWithAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Nd-Username"); got != "admin" {
			t.Fatalf("X-Nd-Username = %q, want admin", got)
		}
		if got := r.Header.Get("X-Nd-Apikey"); got != "api-key" {
			t.Fatalf("X-Nd-Apikey = %q, want api-key", got)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), ConnectionConfig{
		Endpoint: server.URL,
		Username: "admin",
		APIKey:   "api-key",
	})

	if _, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/infra/myRoles",
	}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestClientDoWithPasswordAuthLogsIn(t *testing.T) {
	t.Parallel()

	var loginCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			loginCalls++
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			_, _ = w.Write([]byte(`{"token":"session-token"}`))
		case "/api/v1/infra/myRoles":
			if got := r.Header.Get("Authorization"); got != "Bearer session-token" {
				t.Fatalf("Authorization = %q, want Bearer session-token", got)
			}
			_, _ = w.Write([]byte(`{"roles":["super-admin"]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), ConnectionConfig{
		Endpoint: server.URL,
		Username: "admin",
		Password: "secret",
		Domain:   "local",
	})

	if _, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/api/v1/infra/myRoles",
	}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if loginCalls != 1 {
		t.Fatalf("loginCalls = %d, want 1", loginCalls)
	}
}
