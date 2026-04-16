package webex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestClientDoSuccessAndBearerAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		if r.URL.Path != "/v1/people/me" {
			t.Fatalf("Path = %q, want /v1/people/me", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"person-1","displayName":"Example"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), server.URL+"/v1", "test-token")
	result, err := client.Do(context.Background(), contracts.OperationDescriptor{
		Method: "GET",
		Path:   "/people/me",
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Do() result type = %T, want map[string]any", result)
	}
	if got := payload["id"]; got != "person-1" {
		t.Fatalf("id = %#v, want person-1", got)
	}
}
