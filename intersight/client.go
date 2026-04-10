package intersight

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
	"time"

	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	tokens     TokenProvider
}

func NewClient(httpClient *http.Client, baseURL string, tokens TokenProvider) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		tokens:     tokens,
	}
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if c.tokens == nil {
		return nil, contracts.InternalError{Message: "token provider is not configured"}
	}

	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, err
	}

	endpoint := operation.EndpointURL
	if endpoint == "" {
		endpoint = joinEndpoint(c.baseURL, requestPath)
	}
	u, err := url.Parse(endpoint)
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

	var bodyReader io.Reader
	if operation.Body != nil {
		payload, err := json.Marshal(operation.Body)
		if err != nil {
			return nil, contracts.ValidationError{Message: "request body is not valid JSON", Err: err}
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, operation.Method, u.String(), bodyReader)
	if err != nil {
		return nil, contracts.InternalError{Message: "build Intersight request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if operation.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range operation.Headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       u.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, normalizeTransportError(err, ctx, "Intersight request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         u.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, contracts.NetworkError{Message: "read Intersight response failed", Err: readErr}
	}
	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         u.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Intersight returned HTTP %d", resp.StatusCode),
		}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.NetworkError{Message: "response body was not valid JSON", Err: err}
	}
	return decoded, nil
}

func (c *Client) DoJSON(ctx context.Context, method, path string, options RequestOptions) (any, error) {
	operation := contracts.NewHTTPOperationDescriptor(method, path)
	operation.QueryParams = cloneStringSliceMap(options.Query)
	operation.Headers = cloneStringSliceMap(options.Headers)
	operation.Body = options.Body
	operation.EndpointURL = options.EndpointURL
	return c.Do(ctx, operation)
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
	case strings.HasPrefix(path, "/api/v1/") && strings.HasSuffix(base, "/api/v1"):
		return strings.TrimSuffix(base, "/api/v1") + path
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

func cloneStringSliceMap(in map[string]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = []string{v}
	}
	return out
}
