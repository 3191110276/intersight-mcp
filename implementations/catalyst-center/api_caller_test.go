package catalystcenter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestClientDoUsesStaticToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Auth-Token"); got != "static-token" {
			t.Fatalf("X-Auth-Token = %q, want static-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewClient(server.Client(), ConnectionConfig{
		Endpoint:    server.URL,
		StaticToken: "static-token",
	})
	result, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/dna/intent/api/v1/site",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	decoded, ok := result.(map[string]any)
	if !ok || decoded["ok"] != true {
		t.Fatalf("Do() result = %#v, want ok=true", result)
	}
}

func TestClientDoFetchesManagedToken(t *testing.T) {
	t.Parallel()

	var issued atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case authTokenPath:
			if got := r.Header.Get("Authorization"); got == "" {
				t.Fatalf("Authorization header is empty")
			}
			issued.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"Token": "token-1"})
		case "/dna/intent/api/v1/site":
			if got := r.Header.Get("X-Auth-Token"); got != "token-1" {
				t.Fatalf("X-Auth-Token = %q, want token-1", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"response":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), ConnectionConfig{
		Endpoint: server.URL,
		Username: "user",
		Password: "pass",
	})
	result, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method:       http.MethodGet,
		PathTemplate: "/dna/intent/api/v1/site",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if issued.Load() != 1 {
		t.Fatalf("token fetch count = %d, want 1", issued.Load())
	}
	decoded, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Do() result type = %T, want object", result)
	}
	if _, ok := decoded["response"]; !ok {
		t.Fatalf("Do() result = %#v, want response field", result)
	}
}
