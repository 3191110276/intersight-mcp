package thousandeyes

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

const maxThousandEyesResponseBytes = 16 << 20

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewClient(httpClient *http.Client, baseURL, token string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      strings.TrimSpace(token),
	}
}

func newAPICaller(cfg ConnectionConfig, httpClient *http.Client) *Client {
	return NewClient(httpClient, cfg.Endpoint, cfg.Token)
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, contracts.AuthError{Message: "ThousandEyes bearer token is not configured; search is available, but query and mutate require THOUSANDEYES_BEARER_TOKEN or THOUSANDEYES_API_TOKEN"}
	}

	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(operation.EndpointURL)
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

	var bodyPayload []byte
	if operation.Body != nil {
		bodyPayload, err = json.Marshal(operation.Body)
		if err != nil {
			return nil, contracts.ValidationError{Message: "request body is not valid JSON", Err: err}
		}
	}

	var bodyReader io.Reader
	if bodyPayload != nil {
		bodyReader = bytes.NewReader(bodyPayload)
	}

	req, err := http.NewRequestWithContext(ctx, operation.Method, u.String(), bodyReader)
	if err != nil {
		return nil, contracts.InternalError{Message: "build ThousandEyes request", Err: err}
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
	req.Header.Set("Authorization", "Bearer "+c.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       req.URL.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, normalizeTransportError(err, ctx, "ThousandEyes request failed")
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxThousandEyesResponseBytes+1))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         req.URL.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, contracts.NetworkError{Message: "read ThousandEyes response failed", Err: readErr}
	}
	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         req.URL.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if len(body) > maxThousandEyesResponseBytes {
		return nil, contracts.OutputTooLarge{
			Message: fmt.Sprintf("ThousandEyes response exceeded the %d MiB limit", maxThousandEyesResponseBytes/(1<<20)),
			Details: map[string]any{
				"bytes": len(body),
				"limit": maxThousandEyesResponseBytes,
			},
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, contracts.AuthError{
			Message: fmt.Sprintf("ThousandEyes returned HTTP %d", resp.StatusCode),
			Hint:    "Check THOUSANDEYES_BEARER_TOKEN or THOUSANDEYES_API_TOKEN.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("ThousandEyes returned HTTP %d", resp.StatusCode),
		}
	}

	return decodeThousandEyesJSONBody(body)
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

func decodeThousandEyesJSONBody(body []byte) (any, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.ValidationError{Message: "ThousandEyes returned a non-JSON response", Err: err}
	}
	return decoded, nil
}
