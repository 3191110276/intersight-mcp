package catalystsdwan

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

const maxCatalystSDWANResponseBytes = 16 << 20

type Client struct {
	httpClient *http.Client
	cfg        ConnectionConfig

	mu       sync.Mutex
	cookie   string
	xsrf     string
	loggedIn bool
}

func NewClient(httpClient *http.Client, cfg ConnectionConfig) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		cfg:        cfg,
		cookie:     cfg.SessionCookie(),
		xsrf:       strings.TrimSpace(cfg.XSRFToken),
		loggedIn:   cfg.SessionCookie() != "" || cfg.BearerToken != "",
	}
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	body, err := c.doRequest(ctx, operation, true)
	if err != nil {
		return nil, err
	}
	return decodeCatalystResponseBody(operation, body)
}

func (c *Client) doRequest(ctx context.Context, operation contracts.OperationDescriptor, allowRetry bool) ([]byte, error) {
	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(joinEndpoint(c.cfg.Endpoint, requestPath))
	if err != nil {
		return nil, contracts.ValidationError{Message: "invalid request endpoint", Err: err}
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
			return nil, contracts.ValidationError{Message: "request body is not valid JSON", Err: err}
		}
	}

	respBody, status, err := c.doRequestAttempt(ctx, operation, u.String(), bodyPayload)
	if err == nil {
		return respBody, nil
	}

	if allowRetry && (status == http.StatusUnauthorized || status == http.StatusForbidden) && c.canRefreshSession() {
		if refreshErr := c.resetSession(ctx); refreshErr == nil {
			return c.doRequest(ctx, operation, false)
		}
	}
	return nil, err
}

func (c *Client) doRequestAttempt(ctx context.Context, operation contracts.OperationDescriptor, endpoint string, bodyPayload []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, operation.Method, endpoint, bytes.NewReader(bodyPayload))
	if err != nil {
		return nil, 0, contracts.InternalError{Message: "build Catalyst SD-WAN request", Err: err}
	}

	req.Header.Set("Accept", "application/json")
	if len(bodyPayload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range operation.Headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if err := c.applyAuth(ctx, req); err != nil {
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
		return nil, 0, normalizeTransportError(err, ctx, "Catalyst SD-WAN request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxCatalystSDWANResponseBytes+1))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         req.URL.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, resp.StatusCode, contracts.NetworkError{Message: "read Catalyst SD-WAN response failed", Err: readErr}
	}

	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         req.URL.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if len(body) > maxCatalystSDWANResponseBytes {
		return nil, resp.StatusCode, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Catalyst SD-WAN response exceeded the %d MiB limit", maxCatalystSDWANResponseBytes/(1<<20)),
			Details: map[string]any{
				"bytes": len(body),
				"limit": maxCatalystSDWANResponseBytes,
			},
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, resp.StatusCode, contracts.AuthError{
			Message: fmt.Sprintf("Catalyst SD-WAN returned HTTP %d", resp.StatusCode),
			Hint:    "Check CATALYST_SDWAN endpoint and credentials.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Catalyst SD-WAN returned HTTP %d", resp.StatusCode),
		}
	}

	return body, resp.StatusCode, nil
}

func (c *Client) applyAuth(ctx context.Context, req *http.Request) error {
	if token := strings.TrimSpace(c.cfg.BearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		if xsrf := strings.TrimSpace(c.cfg.XSRFToken); xsrf != "" {
			req.Header.Set("X-XSRF-TOKEN", xsrf)
		}
		return nil
	}

	c.mu.Lock()
	cookie := c.cookie
	xsrf := c.xsrf
	loggedIn := c.loggedIn
	c.mu.Unlock()

	if cookie == "" && !loggedIn {
		if err := c.login(ctx); err != nil {
			return err
		}
		c.mu.Lock()
		cookie = c.cookie
		xsrf = c.xsrf
		c.mu.Unlock()
	}

	if cookie == "" {
		return contracts.AuthError{Message: "Catalyst SD-WAN session cookie is not available"}
	}
	req.Header.Set("Cookie", cookie)
	if xsrf != "" {
		req.Header.Set("X-XSRF-TOKEN", xsrf)
	}
	return nil
}

func (c *Client) login(ctx context.Context) error {
	if strings.TrimSpace(c.cfg.Username) == "" || c.cfg.Password == "" {
		return contracts.AuthError{Message: "Catalyst SD-WAN username/password are not configured"}
	}

	c.mu.Lock()
	if c.loggedIn && c.cookie != "" {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	form := url.Values{}
	form.Set("j_username", c.cfg.Username)
	form.Set("j_password", c.cfg.Password)

	loginURL := joinEndpoint(c.cfg.Endpoint, "/j_security_check")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return contracts.InternalError{Message: "build Catalyst SD-WAN login request", Err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return normalizeTransportError(err, ctx, "Catalyst SD-WAN login failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return contracts.NetworkError{Message: "read Catalyst SD-WAN login response failed", Err: readErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return contracts.AuthError{
			Message: fmt.Sprintf("Catalyst SD-WAN login returned HTTP %d", resp.StatusCode),
			Hint:    "Check CATALYST_SDWAN username, password, and endpoint.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if bytes.Contains(bytes.ToLower(body), []byte("<html")) {
		return contracts.AuthError{
			Message: "Catalyst SD-WAN login was rejected",
			Hint:    "Check CATALYST_SDWAN username, password, and endpoint.",
		}
	}

	var cookie string
	for _, candidate := range resp.Cookies() {
		if strings.EqualFold(candidate.Name, "JSESSIONID") {
			cookie = candidate.Name + "=" + candidate.Value
			break
		}
	}
	if cookie == "" {
		return contracts.AuthError{
			Message: "Catalyst SD-WAN login succeeded but did not return a JSESSIONID cookie",
			Hint:    "Check the endpoint and authentication mode.",
		}
	}

	xsrf, err := c.fetchXSRFToken(ctx, cookie)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.cookie = cookie
	c.xsrf = xsrf
	c.loggedIn = true
	c.mu.Unlock()
	return nil
}

func (c *Client) fetchXSRFToken(ctx context.Context, cookie string) (string, error) {
	tokenURL := joinEndpoint(c.cfg.Endpoint, "/dataservice/client/token")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", contracts.InternalError{Message: "build Catalyst SD-WAN XSRF token request", Err: err}
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "text/plain,application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", normalizeTransportError(err, ctx, "Catalyst SD-WAN token request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return "", contracts.NetworkError{Message: "read Catalyst SD-WAN token response failed", Err: readErr}
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", contracts.AuthError{
			Message: fmt.Sprintf("Catalyst SD-WAN token request returned HTTP %d", resp.StatusCode),
			Hint:    "Check CATALYST_SDWAN credentials and endpoint.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	return strings.TrimSpace(string(body)), nil
}

func (c *Client) canRefreshSession() bool {
	return c.cfg.BearerToken == "" && c.cfg.Username != "" && c.cfg.Password != ""
}

func (c *Client) resetSession(ctx context.Context) error {
	c.mu.Lock()
	c.cookie = ""
	c.xsrf = ""
	c.loggedIn = false
	c.mu.Unlock()
	return c.login(ctx)
}

var pathTemplateParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func resolveOperationPath(operation contracts.OperationDescriptor) (string, error) {
	if strings.TrimSpace(operation.Path) != "" {
		return operation.Path, nil
	}

	template := strings.TrimSpace(operation.PathTemplate)
	if template == "" {
		return "", contracts.ValidationError{Message: "request path is required"}
	}

	missing := []string{}
	resolved := pathTemplateParamPattern.ReplaceAllStringFunc(template, func(segment string) string {
		matches := pathTemplateParamPattern.FindStringSubmatch(segment)
		if len(matches) != 2 {
			return segment
		}
		name := matches[1]
		value, ok := operation.PathParams[name]
		if !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, name)
			return segment
		}
		return url.PathEscape(value)
	})
	if len(missing) > 0 {
		return "", contracts.ValidationError{
			Message: fmt.Sprintf("missing required path params: %s", strings.Join(missing, ", ")),
			Details: map[string]any{"pathTemplate": template, "missing": missing},
		}
	}
	return resolved, nil
}

func joinEndpoint(baseURL, path string) string {
	base := strings.TrimRight(baseURL, "/")
	switch {
	case path == "":
		return base
	case strings.HasPrefix(path, "http://"), strings.HasPrefix(path, "https://"):
		return path
	case strings.HasPrefix(path, "/"):
		return base + path
	default:
		return base + "/" + strings.TrimLeft(path, "/")
	}
}

func normalizeTransportError(err error, ctx context.Context, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return contracts.TimeoutError{Message: message, Err: err}
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return contracts.NetworkError{Message: message, Err: err}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return contracts.TimeoutError{Message: message, Err: err}
	}

	type timeout interface{ Timeout() bool }
	var netTimeout timeout
	if errors.As(err, &netTimeout) && netTimeout.Timeout() {
		return contracts.TimeoutError{Message: message, Err: err}
	}

	return contracts.NetworkError{Message: message, Err: err}
}

func decodeBody(body []byte) any {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err == nil {
		return decoded
	}
	return string(trimmed)
}

func decodeCatalystResponseBody(operation contracts.OperationDescriptor, body []byte) (any, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return map[string]any{}, nil
	}

	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err == nil {
		return decoded, nil
	}
	return string(body), nil
}
