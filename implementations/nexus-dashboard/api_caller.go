package nexusdashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const maxNexusDashboardResponseBytes = 16 << 20

type Client struct {
	httpClient *http.Client
	baseURL    string
	cfg        ConnectionConfig

	mu           sync.Mutex
	sessionToken string
}

func NewClient(httpClient *http.Client, cfg ConnectionConfig) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(cfg.Endpoint, "/"),
		cfg:        cfg,
	}
}

func newAPICaller(cfg ConnectionConfig, httpClient *http.Client) *Client {
	return NewClient(httpClient, cfg)
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if c == nil {
		return nil, contracts.InternalError{Message: "Nexus Dashboard client is nil"}
	}

	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, err
	}

	if isLoginOperation(operation, requestPath) {
		result, _, err := c.doRequest(ctx, operation, authModeNone)
		return result, err
	}

	mode := c.authMode()
	if mode == authModeNone {
		return nil, contracts.AuthError{Message: "Nexus Dashboard credentials are not configured; search is available, but query and mutate require a bearer token, username plus API key, or username plus password"}
	}

	result, status, err := c.doAuthedRequest(ctx, operation, mode)
	if err == nil {
		return result, nil
	}
	if mode == authModePassword && (status == http.StatusUnauthorized || status == http.StatusForbidden) {
		c.clearSessionToken()
		result, _, retryErr := c.doAuthedRequest(ctx, operation, mode)
		return result, retryErr
	}
	return nil, err
}

type requestAuthMode int

const (
	authModeNone requestAuthMode = iota
	authModeToken
	authModeAPIKey
	authModePassword
)

func (c *Client) authMode() requestAuthMode {
	switch {
	case c.cfg.HasToken():
		return authModeToken
	case c.cfg.HasAPIKey():
		return authModeAPIKey
	case c.cfg.HasPasswordAuth():
		return authModePassword
	default:
		return authModeNone
	}
}

func (c *Client) doAuthedRequest(ctx context.Context, operation contracts.OperationDescriptor, mode requestAuthMode) (any, int, error) {
	result, status, err := c.doRequest(ctx, operation, mode)
	if err != nil {
		return nil, status, err
	}
	return result, status, nil
}

func (c *Client) doRequest(ctx context.Context, operation contracts.OperationDescriptor, mode requestAuthMode) (any, int, error) {
	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, 0, err
	}

	endpoint := strings.TrimSpace(operation.EndpointURL)
	if endpoint == "" {
		endpoint = joinEndpoint(c.baseURL, requestPath)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, 0, contracts.ValidationError{Message: "invalid request endpoint", Err: err}
	}
	query := u.Query()
	for key, values := range operation.QueryParams {
		query.Del(key)
		for _, value := range values {
			query.Add(key, value)
		}
	}
	u.RawQuery = query.Encode()

	var bodyPayload []byte
	if operation.Body != nil {
		bodyPayload, err = json.Marshal(operation.Body)
		if err != nil {
			return nil, 0, contracts.ValidationError{Message: "request body is not valid JSON", Err: err}
		}
	}

	var bodyReader io.Reader
	if bodyPayload != nil {
		bodyReader = bytes.NewReader(bodyPayload)
	}

	req, err := http.NewRequestWithContext(ctx, operation.Method, u.String(), bodyReader)
	if err != nil {
		return nil, 0, contracts.InternalError{Message: "build Nexus Dashboard request", Err: err}
	}
	req.Header.Set("Accept", "application/json, application/hal+json, application/problem+json")
	if bodyPayload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range operation.Headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if err := c.applyAuth(ctx, req, mode); err != nil {
		return nil, 0, err
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       req.URL.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, 0, normalizeTransportError(err, ctx, "Nexus Dashboard request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxNexusDashboardResponseBytes+1))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         req.URL.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, resp.StatusCode, contracts.NetworkError{Message: "read Nexus Dashboard response failed", Err: readErr}
	}
	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         req.URL.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if len(body) > maxNexusDashboardResponseBytes {
		return nil, resp.StatusCode, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Nexus Dashboard response exceeded the %d MiB limit", maxNexusDashboardResponseBytes/(1<<20)),
			Details: map[string]any{
				"bytes": len(body),
				"limit": maxNexusDashboardResponseBytes,
			},
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, contracts.AuthError{
			Message: fmt.Sprintf("Nexus Dashboard returned HTTP %d", resp.StatusCode),
			Hint:    "Check NEXUS_DASHBOARD credentials, token, API key, and endpoint.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Nexus Dashboard returned HTTP %d", resp.StatusCode),
		}
	}

	decoded, err := decodeNexusDashboardJSONBody(body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return decoded, resp.StatusCode, nil
}

func (c *Client) applyAuth(ctx context.Context, req *http.Request, mode requestAuthMode) error {
	switch mode {
	case authModeNone:
		return nil
	case authModeToken:
		c.applyTokenHeaders(req, c.cfg.Token)
		return nil
	case authModeAPIKey:
		req.Header.Set("X-Nd-Username", c.cfg.Username)
		req.Header.Set("X-Nd-Apikey", c.cfg.APIKey)
		return nil
	case authModePassword:
		token, err := c.ensureSessionToken(ctx)
		if err != nil {
			return err
		}
		c.applyTokenHeaders(req, token)
		return nil
	default:
		return contracts.InternalError{Message: "unsupported Nexus Dashboard auth mode"}
	}
}

func (c *Client) applyTokenHeaders(req *http.Request, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Cookie", "AuthCookie="+token)
}

func (c *Client) ensureSessionToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	token := strings.TrimSpace(c.sessionToken)
	c.mu.Unlock()
	if token != "" {
		return token, nil
	}

	loginBody := map[string]string{
		"userName":   c.cfg.Username,
		"userPasswd": c.cfg.Password,
		"domain":     c.cfg.Domain,
	}
	bodyPayload, err := json.Marshal(loginBody)
	if err != nil {
		return "", contracts.InternalError{Message: "marshal Nexus Dashboard login payload", Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinEndpoint(c.baseURL, "/login"), bytes.NewReader(bodyPayload))
	if err != nil {
		return "", contracts.InternalError{Message: "build Nexus Dashboard login request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", normalizeTransportError(err, ctx, "Nexus Dashboard login request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxNexusDashboardResponseBytes+1))
	if readErr != nil {
		return "", contracts.NetworkError{Message: "read Nexus Dashboard login response failed", Err: readErr}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", contracts.AuthError{
			Message: "Nexus Dashboard login failed",
			Hint:    "Check NEXUS_DASHBOARD_USERNAME, NEXUS_DASHBOARD_PASSWORD, and NEXUS_DASHBOARD_DOMAIN.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Nexus Dashboard login returned HTTP %d", resp.StatusCode),
		}
	}

	token, err = extractToken(body)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", contracts.AuthError{
			Message: "Nexus Dashboard login response did not contain a token",
			Hint:    "Check NEXUS_DASHBOARD_USERNAME, NEXUS_DASHBOARD_PASSWORD, and NEXUS_DASHBOARD_DOMAIN.",
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(c.sessionToken) == "" {
		c.sessionToken = token
	}
	return c.sessionToken, nil
}

func (c *Client) clearSessionToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionToken = ""
}

func extractToken(body []byte) (string, error) {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", contracts.ValidationError{Message: "Nexus Dashboard login returned a non-JSON response", Err: err}
	}
	for _, key := range []string{"token", "jwttoken"} {
		if token, ok := decoded[key].(string); ok && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token), nil
		}
	}
	return "", nil
}

var pathTemplateParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func resolveOperationPath(operation contracts.OperationDescriptor) (string, error) {
	path := operation.PathTemplate
	if strings.TrimSpace(path) == "" {
		path = operation.Path
	}
	if strings.TrimSpace(path) == "" {
		return "", contracts.ValidationError{Message: "operation path is required"}
	}

	missing := []string{}
	resolved := pathTemplateParamPattern.ReplaceAllStringFunc(path, func(token string) string {
		matches := pathTemplateParamPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		name := matches[1]
		value, ok := operation.PathParams[name]
		if !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, name)
			return token
		}
		return url.PathEscape(value)
	})
	if len(missing) > 0 {
		return "", contracts.ValidationError{
			Message: fmt.Sprintf("missing required path parameters: %s", strings.Join(missing, ", ")),
		}
	}
	if !strings.HasPrefix(resolved, "/") {
		resolved = "/" + resolved
	}
	return resolved, nil
}

func joinEndpoint(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

func normalizeTransportError(err error, ctx context.Context, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return contracts.NetworkError{Message: message, Err: err}
}

func decodeBody(body []byte) any {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		return decoded
	}
	return strings.TrimSpace(string(body))
}

func decodeNexusDashboardJSONBody(body []byte) (any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.ValidationError{Message: "Nexus Dashboard returned a non-JSON response", Err: err}
	}
	return decoded, nil
}

func isLoginOperation(operation contracts.OperationDescriptor, requestPath string) bool {
	if strings.EqualFold(strings.TrimSpace(operation.OperationID), "legacynd_login") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(requestPath), "/login")
}
