package intersight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"golang.org/x/sync/singleflight"
)

const (
	refreshFailureWindow = 5 * time.Minute
	maxRefreshBackoff    = 1 * time.Minute
	minRefreshLead       = 5 * time.Second
	defaultRefreshLead   = 1 * time.Minute
)

type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type OAuthConfig struct {
	TokenURL         string
	ValidateURL      string
	ClientID         string
	ClientSecret     string
	HTTPClient       *http.Client
	Clock            Clock
	BootstrapContext context.Context
}

type Manager struct {
	httpClient   *http.Client
	tokenURL     string
	validateURL  string
	clientID     string
	clientSecret string
	clock        Clock

	refreshGroup singleflight.Group

	mu                    sync.RWMutex
	token                 accessToken
	refreshFailures       []time.Time
	refreshBackoff        time.Duration
	nextRefreshAttempt    time.Time
	degraded              bool
	degradedUntil         time.Time
	proactiveRefreshAlive bool
}

type accessToken struct {
	Value     string
	ExpiresAt time.Time
	RefreshAt time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func NewOAuthManager(ctx context.Context, cfg OAuthConfig) (*Manager, error) {
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, contracts.AuthError{Message: "missing required INTERSIGHT_CLIENT_ID"}
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, contracts.AuthError{Message: "missing required INTERSIGHT_CLIENT_SECRET"}
	}
	if strings.TrimSpace(cfg.TokenURL) == "" {
		return nil, contracts.AuthError{Message: "missing OAuth token URL"}
	}
	if strings.TrimSpace(cfg.ValidateURL) == "" {
		return nil, contracts.AuthError{Message: "missing OAuth validation URL"}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	clock := cfg.Clock
	if clock == nil {
		clock = realClock{}
	}
	bootstrapCtx := cfg.BootstrapContext
	if bootstrapCtx == nil {
		bootstrapCtx = ctx
	}

	m := &Manager{
		httpClient:            httpClient,
		tokenURL:              cfg.TokenURL,
		validateURL:           cfg.ValidateURL,
		clientID:              cfg.ClientID,
		clientSecret:          cfg.ClientSecret,
		clock:                 clock,
		proactiveRefreshAlive: true,
	}

	if _, err := m.refreshToken(bootstrapCtx, true); err != nil {
		return nil, err
	}
	go m.refreshLoop(ctx)
	return m, nil
}

func (m *Manager) Token(ctx context.Context) (string, error) {
	if err := m.ensureAvailable(ctx); err != nil {
		return "", err
	}

	m.mu.RLock()
	token := m.token
	m.mu.RUnlock()
	if token.Value == "" {
		return "", contracts.AuthError{Message: "OAuth token unavailable"}
	}
	now := m.clock.Now()
	if m.shouldRefresh(token, now) {
		refreshed, err := m.refreshToken(ctx, false)
		if err != nil {
			if m.tokenStillUsable(token, now) && !m.IsDegraded() {
				return token.Value, nil
			}
			return "", err
		}
		token = refreshed
	}
	return token.Value, nil
}

func (m *Manager) IsDegraded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.degraded
}

func (m *Manager) ensureAvailable(ctx context.Context) error {
	m.mu.RLock()
	degraded := m.degraded
	degradedUntil := m.degradedUntil
	m.mu.RUnlock()

	if !degraded {
		return nil
	}
	if m.clock.Now().Before(degradedUntil) {
		return contracts.AuthError{Message: "authentication is in degraded mode after repeated token refresh failures"}
	}
	if _, err := m.refreshToken(ctx, false); err != nil {
		return err
	}
	return nil
}

func (m *Manager) refreshLoop(ctx context.Context) {
	for {
		wait := m.nextRefreshDelay()
		select {
		case <-ctx.Done():
			return
		case <-m.clock.After(wait):
		}

		m.mu.RLock()
		degraded := m.degraded
		proactiveAlive := m.proactiveRefreshAlive
		m.mu.RUnlock()
		if !proactiveAlive || degraded {
			continue
		}
		_, _ = m.refreshToken(ctx, false)
	}
}

func (m *Manager) nextRefreshDelay() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.degraded || !m.proactiveRefreshAlive {
		return refreshFailureWindow
	}
	if m.token.Value == "" {
		return m.delayUntilRefreshAllowed(m.clock.Now())
	}

	now := m.clock.Now()
	wait := m.token.RefreshAt.Sub(now)
	if backoffWait := m.delayUntilRefreshAllowed(now); backoffWait > wait {
		wait = backoffWait
	}
	if wait < 0 {
		return 0
	}
	return wait
}

func (m *Manager) shouldRefresh(token accessToken, now time.Time) bool {
	if token.Value == "" {
		return true
	}
	if token.RefreshAt.IsZero() {
		return !now.Before(token.ExpiresAt)
	}
	return !now.Before(token.RefreshAt)
}

func (m *Manager) tokenStillUsable(token accessToken, now time.Time) bool {
	return token.Value != "" && now.Before(token.ExpiresAt)
}

func refreshLeadTime(lifetime time.Duration) time.Duration {
	if lifetime <= 0 {
		return 0
	}
	if lifetime <= 2*minRefreshLead {
		return lifetime / 2
	}
	if lifetime < defaultRefreshLead {
		return minRefreshLead
	}
	return defaultRefreshLead
}

func (m *Manager) refreshToken(ctx context.Context, validate bool) (accessToken, error) {
	if wait := m.delayUntilRefreshAllowed(m.clock.Now()); wait > 0 {
		return accessToken{}, contracts.AuthError{Message: fmt.Sprintf("token refresh is backing off after repeated failures; retry in %.0fs", wait.Seconds())}
	}

	result, err, _ := m.refreshGroup.Do("oauth-refresh", func() (any, error) {
		token, err := m.fetchToken(ctx)
		if err != nil {
			m.recordRefreshFailure()
			return accessToken{}, err
		}
		if validate {
			if err := m.validateToken(ctx, token.Value); err != nil {
				m.recordRefreshFailure()
				return accessToken{}, err
			}
		}
		m.recordRefreshSuccess(token)
		return token, nil
	})
	if err != nil {
		return accessToken{}, err
	}
	return result.(accessToken), nil
}

func (m *Manager) fetchToken(ctx context.Context) (accessToken, error) {
	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", m.clientID)
	values.Set("client_secret", m.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return accessToken{}, contracts.InternalError{Message: "build OAuth token request", Err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return accessToken{}, normalizeTransportError(err, ctx, "OAuth token request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return accessToken{}, contracts.NetworkError{Message: "read OAuth token response failed", Err: readErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return accessToken{}, contracts.AuthError{
			Message: fmt.Sprintf("OAuth token request failed with HTTP %d", resp.StatusCode),
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}

	var payload tokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return accessToken{}, contracts.AuthError{Message: "OAuth token response was not valid JSON", Err: err}
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return accessToken{}, contracts.AuthError{Message: "OAuth token response did not include access_token"}
	}
	if payload.ExpiresIn <= 0 {
		return accessToken{}, contracts.AuthError{Message: "OAuth token response did not include a positive expires_in"}
	}

	issuedAt := m.clock.Now()
	lifetime := time.Duration(payload.ExpiresIn) * time.Second
	return accessToken{
		Value:     payload.AccessToken,
		ExpiresAt: issuedAt.Add(lifetime),
		RefreshAt: issuedAt.Add(lifetime - refreshLeadTime(lifetime)),
	}, nil
}

func (m *Manager) validateToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.validateURL, nil)
	if err != nil {
		return contracts.InternalError{Message: "build OAuth validation request", Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return contracts.AuthError{Message: "initial token validation failed", Err: normalizeTransportError(err, ctx, "initial token validation failed")}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return contracts.AuthError{Message: "read token validation response failed", Err: readErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return contracts.AuthError{
			Message: fmt.Sprintf("initial token validation failed with HTTP %d", resp.StatusCode),
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	return nil
}

func (m *Manager) recordRefreshSuccess(token accessToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
	m.refreshFailures = nil
	m.refreshBackoff = 0
	m.nextRefreshAttempt = time.Time{}
	m.degraded = false
	m.degradedUntil = time.Time{}
	m.proactiveRefreshAlive = true
}

func (m *Manager) recordRefreshFailure() {
	now := m.clock.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := now.Add(-refreshFailureWindow)
	failures := m.refreshFailures[:0]
	for _, ts := range m.refreshFailures {
		if !ts.Before(cutoff) {
			failures = append(failures, ts)
		}
	}
	failures = append(failures, now)
	m.refreshFailures = failures

	if m.refreshBackoff == 0 {
		m.refreshBackoff = time.Second
	} else {
		next := time.Duration(math.Min(float64(m.refreshBackoff*2), float64(maxRefreshBackoff)))
		m.refreshBackoff = next
	}
	m.nextRefreshAttempt = now.Add(m.refreshBackoff)

	if len(m.refreshFailures) >= 3 {
		m.degraded = true
		m.degradedUntil = now.Add(refreshFailureWindow)
		m.proactiveRefreshAlive = false
	}
}

func (m *Manager) currentBackoff() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.refreshBackoff
}

func (m *Manager) delayUntilRefreshAllowed(now time.Time) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.nextRefreshAttempt.IsZero() {
		return 0
	}
	if now.Before(m.nextRefreshAttempt) {
		return m.nextRefreshAttempt.Sub(now)
	}
	return 0
}

type RequestOptions struct {
	Query       map[string]string
	Body        any
	Headers     map[string]string
	EndpointURL string
}
