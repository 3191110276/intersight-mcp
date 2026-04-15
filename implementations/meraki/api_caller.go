package meraki

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
	"strconv"
	"strings"
	"time"

	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const maxMerakiResponseBytes = 16 << 20
const (
	merakiRetryAttempts     = 3
	merakiInitialBackoff    = 500 * time.Millisecond
	merakiPaginationMaxPage = 100
)

type unavailableClient struct {
	err error
}

func (c unavailableClient) Do(_ context.Context, _ contracts.OperationDescriptor) (any, error) {
	return nil, c.err
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewClient(httpClient *http.Client, baseURL, apiKey string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
	}
}

func newAPICaller(cfg ConnectionConfig, httpClient *http.Client) *Client {
	return NewClient(httpClient, cfg.Endpoint, cfg.APIKey)
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, contracts.AuthError{Message: "Meraki API key is not configured; search is available, but query and mutate require MERAKI_API_KEY or MERAKI_DASHBOARD_API_KEY"}
	}

	if operation.FollowUpPlan.Kind == merakiListAllFollowUpKind {
		return c.doListAll(ctx, operation)
	}

	return c.doSingle(ctx, operation)
}

func (c *Client) doSingle(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	body, _, err := c.doRequest(ctx, operation)
	if err != nil {
		return nil, err
	}
	return decodeMerakiJSONBody(body)
}

func (c *Client) doListAll(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	items := []any{}
	current := operation
	pageCount := 0

	for {
		pageCount++
		if pageCount > merakiPaginationMaxPage {
			return nil, contracts.OutputTooLarge{
				Message: fmt.Sprintf("Meraki pagination helper exceeded the %d-page safety limit", merakiPaginationMaxPage),
				Details: map[string]any{"pages": pageCount - 1, "limit": merakiPaginationMaxPage},
			}
		}

		body, headers, err := c.doRequest(ctx, current)
		if err != nil {
			return nil, err
		}

		decoded, err := decodeMerakiJSONBody(body)
		if err != nil {
			return nil, err
		}

		pageItems, ok := decoded.([]any)
		if !ok {
			return nil, contracts.ValidationError{
				Message: "Meraki pagination helper expected an array response",
				Details: map[string]any{
					"operationId": operation.OperationID,
					"path":        operation.PathTemplate,
				},
			}
		}
		items = append(items, pageItems...)

		nextURL, ok := merakiNextLink(headers)
		if !ok {
			return items, nil
		}
		current.EndpointURL = joinEndpoint(c.baseURL, nextURL)
		current.QueryParams = nil
	}
}

func (c *Client) doRequest(ctx context.Context, operation contracts.OperationDescriptor) ([]byte, http.Header, error) {
	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, nil, err
	}

	endpoint := strings.TrimSpace(operation.EndpointURL)
	if endpoint == "" {
		endpoint = joinEndpoint(c.baseURL, requestPath)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, contracts.ValidationError{Message: "invalid request endpoint", Err: err}
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
			return nil, nil, contracts.ValidationError{Message: "request body is not valid JSON", Err: err}
		}
	}

	var lastErr error
	for attempt := 1; attempt <= merakiRetryAttempts; attempt++ {
		body, headers, retryDelay, retryable, err := c.doRequestAttempt(ctx, operation, u.String(), bodyPayload, attempt)
		if err == nil {
			return body, headers, nil
		}
		lastErr = err
		if !retryable || attempt == merakiRetryAttempts {
			return nil, nil, err
		}
		if waitErr := merakiSleepWithContext(ctx, retryDelay); waitErr != nil {
			return nil, nil, waitErr
		}
	}
	return nil, nil, lastErr
}

func (c *Client) doRequestAttempt(ctx context.Context, operation contracts.OperationDescriptor, endpoint string, bodyPayload []byte, attempt int) ([]byte, http.Header, time.Duration, bool, error) {
	var bodyReader io.Reader
	if bodyPayload != nil {
		bodyReader = bytes.NewReader(bodyPayload)
	}

	req, err := http.NewRequestWithContext(ctx, operation.Method, endpoint, bodyReader)
	if err != nil {
		return nil, nil, 0, false, contracts.InternalError{Message: "build Meraki request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if bodyPayload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range operation.Headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       req.URL.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, nil, 0, false, normalizeTransportError(err, ctx, "Meraki request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxMerakiResponseBytes+1))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         req.URL.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, nil, 0, false, contracts.NetworkError{Message: "read Meraki response failed", Err: readErr}
	}
	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         req.URL.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if len(body) > maxMerakiResponseBytes {
		return nil, nil, 0, false, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Meraki response exceeded the %d MiB limit", maxMerakiResponseBytes/(1<<20)),
			Details: map[string]any{
				"bytes": len(body),
				"limit": maxMerakiResponseBytes,
			},
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := merakiRetryDelay(resp.Header, attempt)
		return nil, nil, retryAfter, true, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Meraki rate limit exceeded after %d attempt(s)", merakiRetryAttempts),
		}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, 0, false, contracts.AuthError{
			Message: fmt.Sprintf("Meraki returned HTTP %d", resp.StatusCode),
			Hint:    "Check MERAKI_API_KEY or MERAKI_DASHBOARD_API_KEY.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, 0, false, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Meraki returned HTTP %d", resp.StatusCode),
		}
	}

	return body, resp.Header.Clone(), 0, false, nil
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
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return contracts.TimeoutError{Message: message, Err: err}
		}
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

func decodeMerakiJSONBody(body []byte) (any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.NetworkError{Message: "response body was not valid JSON", Err: err}
	}
	return decoded, nil
}

func merakiRetryDelay(header http.Header, attempt int) time.Duration {
	if value := strings.TrimSpace(header.Get("Retry-After")); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		if when, err := http.ParseTime(value); err == nil {
			if delay := time.Until(when); delay > 0 {
				return delay
			}
		}
	}
	delay := merakiInitialBackoff * time.Duration(attempt)
	if delay <= 0 {
		return merakiInitialBackoff
	}
	return delay
}

func merakiSleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return contracts.TimeoutError{Message: "Meraki request timed out while waiting to retry", Err: ctx.Err()}
		}
		return contracts.NetworkError{Message: "Meraki request canceled while waiting to retry", Err: ctx.Err()}
	case <-timer.C:
		return nil
	}
}

func merakiNextLink(header http.Header) (string, bool) {
	raw := strings.TrimSpace(header.Get("Link"))
	if raw == "" {
		return "", false
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start < 0 || end <= start+1 {
			continue
		}
		return part[start+1 : end], true
	}
	return "", false
}
