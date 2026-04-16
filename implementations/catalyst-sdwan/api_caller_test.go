package catalystsdwan

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestClientSessionLoginAndRequest(t *testing.T) {
	var sawCookie bool
	var sawXSRF bool

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/j_security_check":
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "abc123"})
			w.WriteHeader(http.StatusOK)
		case "/dataservice/client/token":
			if got := r.Header.Get("Cookie"); got != "JSESSIONID=abc123" {
				t.Fatalf("token cookie = %q, want JSESSIONID=abc123", got)
			}
			_, _ = io.WriteString(w, "xsrf-token")
		case "/dataservice/device":
			sawCookie = r.Header.Get("Cookie") == "JSESSIONID=abc123"
			sawXSRF = r.Header.Get("X-XSRF-TOKEN") == "xsrf-token"
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":[{"host-name":"edge1"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := ConnectionConfig{
		Endpoint: server.URL,
		Username: "admin",
		Password: "secret",
	}

	api := NewClient(server.Client(), cfg)
	result, err := api.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/dataservice/device",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if !sawCookie {
		t.Fatal("expected request cookie header")
	}
	if !sawXSRF {
		t.Fatal("expected X-XSRF-TOKEN header")
	}
	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", result)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("result = %#v, want data field", payload)
	}
}

func TestClientReturnsPlainTextWhenJSONDecodeFails(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "system\n host-name edge1\n")
	}))
	defer server.Close()

	cfg := ConnectionConfig{
		Endpoint:         server.URL,
		SessionCookieRaw: "JSESSIONID=abc123",
	}

	api := NewClient(server.Client(), cfg)
	result, err := api.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/dataservice/device/config",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if got, ok := result.(string); !ok || got != "system\n host-name edge1\n" {
		t.Fatalf("result = %#v, want plain text response", result)
	}
}
