package intersight

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/testutil"
)

func TestNewOAuthManagerValidatesInitialToken(t *testing.T) {
	t.Parallel()

	var sawValidation atomic.Bool
	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": "bootstrap-token",
				"expires_in":   3600,
			})
		case "/api/v1/iam/UserPreferences":
			if got := r.Header.Get("Authorization"); got != "Bearer bootstrap-token" {
				t.Fatalf("unexpected auth header: %q", got)
			}
			sawValidation.Store(true)
			writeJSON(t, w, http.StatusOK, map[string]any{"Results": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
		Clock:        testutil.NewManualClock(time.Unix(0, 0)),
	})
	if err != nil {
		t.Fatalf("NewOAuthManager() error = %v", err)
	}

	token, err := manager.Token(ctx)
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "bootstrap-token" {
		t.Fatalf("unexpected token: %q", token)
	}
	if !sawValidation.Load() {
		t.Fatalf("expected startup validation request")
	}
}

func TestNewOAuthManagerFailsWhenInitialValidationFails(t *testing.T) {
	t.Parallel()

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": "bootstrap-token",
				"expires_in":   3600,
			})
		case "/api/v1/iam/UserPreferences":
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{"message": "unauthorized"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
	})
	if err == nil {
		t.Fatalf("expected startup validation failure")
	}

	var authErr contracts.AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthError, got %T", err)
	}
}

func TestOAuthRefreshSynchronization(t *testing.T) {
	t.Parallel()

	clock := testutil.NewManualClock(time.Unix(0, 0))
	var tokenCalls atomic.Int32

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			call := tokenCalls.Add(1)
			token := "initial-token"
			if call > 1 {
				token = "refreshed-token"
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": token,
				"expires_in":   8,
			})
		case "/api/v1/iam/UserPreferences":
			writeJSON(t, w, http.StatusOK, map[string]any{"Results": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("NewOAuthManager() error = %v", err)
	}

	clock.Advance(8 * time.Second)

	var wg sync.WaitGroup
	results := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := manager.Token(ctx)
			if err != nil {
				t.Errorf("Token() error = %v", err)
				return
			}
			results <- token
		}()
	}
	wg.Wait()
	close(results)

	for token := range results {
		if token != "refreshed-token" && token != "initial-token" {
			t.Fatalf("unexpected token value: %q", token)
		}
	}
	if got := tokenCalls.Load(); got != 2 {
		t.Fatalf("expected one bootstrap token request and one shared refresh, got %d", got)
	}
}

func TestOAuthDegradedModeAndRecovery(t *testing.T) {
	t.Parallel()

	clock := testutil.NewManualClock(time.Unix(0, 0))
	var tokenCalls atomic.Int32
	var failRefresh atomic.Bool
	failRefresh.Store(true)

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			call := tokenCalls.Add(1)
			if call == 1 || !failRefresh.Load() {
				writeJSON(t, w, http.StatusOK, map[string]any{
					"access_token": "token-ok",
					"expires_in":   8,
				})
				return
			}
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{"message": "bad credentials"})
		case "/api/v1/iam/UserPreferences":
			writeJSON(t, w, http.StatusOK, map[string]any{"Results": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("NewOAuthManager() error = %v", err)
	}

	clock.Advance(8 * time.Second)
	if _, err := manager.Token(ctx); err == nil {
		t.Fatalf("expected first refresh failure")
	}

	clock.Advance(1 * time.Second)
	if _, err := manager.Token(ctx); err == nil {
		t.Fatalf("expected second refresh failure")
	}

	clock.Advance(2 * time.Second)
	if _, err := manager.Token(ctx); err == nil {
		t.Fatalf("expected third refresh failure")
	}
	if !manager.IsDegraded() {
		t.Fatalf("expected degraded mode after three failures")
	}

	if _, err := manager.Token(ctx); err == nil {
		t.Fatalf("expected degraded auth error during cooldown")
	}

	failRefresh.Store(false)
	clock.Advance(5*time.Minute + 1*time.Second)

	token, err := manager.Token(ctx)
	if err != nil {
		t.Fatalf("expected recovery after cooldown, got %v", err)
	}
	if token != "token-ok" {
		t.Fatalf("unexpected recovered token: %q", token)
	}
	if manager.IsDegraded() {
		t.Fatalf("expected degraded mode to clear after recovery")
	}
}

func TestOAuthUsesExistingTokenDuringRefreshBackoff(t *testing.T) {
	t.Parallel()

	clock := testutil.NewManualClock(time.Unix(0, 0))
	var tokenCalls atomic.Int32
	var failRefresh atomic.Bool

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			call := tokenCalls.Add(1)
			if call == 1 || !failRefresh.Load() {
				writeJSON(t, w, http.StatusOK, map[string]any{
					"access_token": "token-ok",
					"expires_in":   120,
				})
				return
			}
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{"message": "bad credentials"})
		case "/api/v1/iam/UserPreferences":
			writeJSON(t, w, http.StatusOK, map[string]any{"Results": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("NewOAuthManager() error = %v", err)
	}

	failRefresh.Store(true)
	clock.Advance(119*time.Second + 900*time.Millisecond)

	token, err := manager.Token(ctx)
	if err != nil {
		t.Fatalf("Token() with valid cached token error = %v", err)
	}
	if token != "token-ok" {
		t.Fatalf("unexpected token after refresh failure: %q", token)
	}

	token, err = manager.Token(ctx)
	if err != nil {
		t.Fatalf("Token() during refresh backoff error = %v", err)
	}
	if token != "token-ok" {
		t.Fatalf("unexpected token during backoff: %q", token)
	}

	if got := tokenCalls.Load(); got != 2 {
		t.Fatalf("expected one bootstrap request and one failed refresh, got %d", got)
	}
}

func TestOAuthUsesExistingTokenDuringDegradedCooldownIfStillValid(t *testing.T) {
	t.Parallel()

	clock := testutil.NewManualClock(time.Unix(0, 0))
	var tokenCalls atomic.Int32
	var failRefresh atomic.Bool

	server := testutil.NewTCP4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/iam/token":
			call := tokenCalls.Add(1)
			if call == 1 || !failRefresh.Load() {
				writeJSON(t, w, http.StatusOK, map[string]any{
					"access_token": "token-ok",
					"expires_in":   120,
				})
				return
			}
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{"message": "bad credentials"})
		case "/api/v1/iam/UserPreferences":
			writeJSON(t, w, http.StatusOK, map[string]any{"Results": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager, err := NewOAuthManager(ctx, OAuthConfig{
		TokenURL:     server.URL + "/iam/token",
		ValidateURL:  server.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   server.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("NewOAuthManager() error = %v", err)
	}

	failRefresh.Store(true)
	clock.Advance(60 * time.Second)
	if token, err := manager.Token(ctx); err != nil || token != "token-ok" {
		t.Fatalf("first refresh failure should still return cached token, got token=%q err=%v", token, err)
	}

	clock.Advance(1 * time.Second)
	if token, err := manager.Token(ctx); err != nil || token != "token-ok" {
		t.Fatalf("second refresh failure should still return cached token, got token=%q err=%v", token, err)
	}

	clock.Advance(2 * time.Second)
	token, err := manager.Token(ctx)
	if err != nil {
		t.Fatalf("degraded cooldown should still return cached token while it is valid, got err=%v", err)
	}
	if token != "token-ok" {
		t.Fatalf("unexpected token during degraded cooldown: %q", token)
	}
	if !manager.IsDegraded() {
		t.Fatalf("expected degraded mode after three failures")
	}

	if got := tokenCalls.Load(); got != 4 {
		t.Fatalf("expected one bootstrap request and three failed refreshes, got %d", got)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}
