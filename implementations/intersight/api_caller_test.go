package intersight

import (
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
	ciscointersight "github.com/mimaurer/intersight-mcp/intersight"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRetryingBootstrapClientUsesRequestContextForBootstrap(t *testing.T) {
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

	client := NewRetryingBootstrapClient(
		context.Background(),
		time.Second,
		api.Client(),
		api.URL+"/api/v1",
		ciscointersight.OAuthConfig{
			TokenURL:     api.URL + "/iam/token",
			ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
			ClientID:     "id",
			ClientSecret: "secret",
			HTTPClient:   api.Client(),
		},
	)

	requestCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.EnsureClient(requestCtx)
	if err == nil {
		t.Fatalf("expected bootstrap error")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("EnsureClient() took %v, want request-bounded timeout", elapsed)
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
	_, err := BootstrapOAuthManager(context.Background(), context.Background(), 50*time.Millisecond, ciscointersight.OAuthConfig{
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
		t.Fatalf("BootstrapOAuthManager() took %v, want bounded timeout", elapsed)
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

	manager, err := BootstrapOAuthManager(ctx, ctx, time.Second, ciscointersight.OAuthConfig{
		TokenURL:     api.URL + "/iam/token",
		ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   api.Client(),
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("BootstrapOAuthManager() error = %v", err)
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

func TestRetryingBootstrapClientHonorsRequestContextDuringBootstrap(t *testing.T) {
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

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	client := NewRetryingBootstrapClient(
		serverCtx,
		time.Second,
		api.Client(),
		api.URL+"/api/v1",
		ciscointersight.OAuthConfig{
			TokenURL:     api.URL + "/iam/token",
			ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
			ClientID:     "id",
			ClientSecret: "secret",
			HTTPClient:   api.Client(),
		},
	)

	requestCtx, requestCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer requestCancel()

	start := time.Now()
	_, err := client.Do(requestCtx, contracts.NewHTTPOperationDescriptor(http.MethodGet, "/api/v1/compute/RackUnits"))
	if err == nil {
		t.Fatal("expected bootstrap retry failure")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("bootstrap retry took %v, want request-bound cancellation", elapsed)
	}
}

func TestRetryingBootstrapClientWaitersCanTimeOutIndependently(t *testing.T) {
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

	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	client := NewRetryingBootstrapClient(
		serverCtx,
		time.Second,
		api.Client(),
		api.URL+"/api/v1",
		ciscointersight.OAuthConfig{
			TokenURL:     api.URL + "/iam/token",
			ValidateURL:  api.URL + "/api/v1/iam/UserPreferences",
			ClientID:     "id",
			ClientSecret: "secret",
			HTTPClient:   api.Client(),
		},
	)

	firstCtx, firstCancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer firstCancel()

	firstErrCh := make(chan error, 1)
	go func() {
		_, err := client.Do(firstCtx, contracts.NewHTTPOperationDescriptor(http.MethodGet, "/api/v1/compute/RackUnits"))
		firstErrCh <- err
	}()

	time.Sleep(25 * time.Millisecond)

	secondCtx, secondCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer secondCancel()

	start := time.Now()
	_, err := client.Do(secondCtx, contracts.NewHTTPOperationDescriptor(http.MethodGet, "/api/v1/compute/RackUnits"))
	if err == nil {
		t.Fatal("expected second bootstrap retry failure")
	}

	var timeoutErr contracts.TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError for waiting caller, got %T", err)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("waiting caller took %v, want independent timeout", elapsed)
	}

	serverCancel()

	select {
	case err := <-firstErrCh:
		if err == nil {
			t.Fatal("expected first bootstrap retry failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first bootstrap attempt to finish")
	}
}

func TestRetryingBootstrapClientBacksOffAfterFailure(t *testing.T) {
	t.Parallel()

	var tokenCalls atomic.Int32
	client := &RetryingBootstrapClient{
		ctx:            context.Background(),
		timeout:        time.Second,
		httpClient:     &http.Client{},
		baseURL:        "https://example.com/api/v1",
		initialBackoff: 100 * time.Millisecond,
		maxBackoff:     time.Second,
		now:            time.Now,
		oauthCfg: ciscointersight.OAuthConfig{
			TokenURL:     "https://example.com/iam/token",
			ValidateURL:  "https://example.com/api/v1/iam/UserPreferences",
			ClientID:     "id",
			ClientSecret: "secret",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				tokenCalls.Add(1)
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"message":"bad credentials"}`)),
					Request:    req,
				}, nil
			})},
		},
	}

	_, err := client.ensureClient(context.Background())
	if err == nil {
		t.Fatal("expected bootstrap error")
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls after first attempt = %d, want 1", got)
	}

	_, err = client.ensureClient(context.Background())
	if err == nil {
		t.Fatal("expected cached bootstrap error during backoff")
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls during backoff = %d, want 1", got)
	}

	time.Sleep(125 * time.Millisecond)

	_, err = client.ensureClient(context.Background())
	if err == nil {
		t.Fatal("expected bootstrap error after backoff retry")
	}
	if got := tokenCalls.Load(); got != 2 {
		t.Fatalf("token calls after backoff expiry = %d, want 2", got)
	}
}
