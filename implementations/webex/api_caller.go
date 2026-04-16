package webex

import (
	"bytes"
	"context"
	"encoding/json"
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

const maxWebexResponseBytes = 16 << 20

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
	return NewClient(httpClient, cfg.Endpoint, cfg.AccessToken)
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, contracts.AuthError{Message: "Webex access token is not configured; query and mutate require WEBEX_ACCESS_TOKEN or WEBEX_BEARER_TOKEN"}
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
		return nil, contracts.InternalError{Message: "build Webex request", Err: err}
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
	req.Header.Set("Authorization", "Bearer "+c.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       req.URL.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, contracts.NetworkError{Message: "Webex request failed", Err: err}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxWebexResponseBytes+1))
	if readErr != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:       operation.Method,
			Path:         req.URL.Path,
			Status:       resp.StatusCode,
			ResponseSize: len(body),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return nil, contracts.NetworkError{Message: "read Webex response failed", Err: readErr}
	}
	internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
		Method:       operation.Method,
		Path:         req.URL.Path,
		Status:       resp.StatusCode,
		ResponseSize: len(body),
		DurationMS:   time.Since(start).Milliseconds(),
	})

	if len(body) > maxWebexResponseBytes {
		return nil, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Webex response exceeded the %d MiB limit", maxWebexResponseBytes/(1<<20)),
		}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, contracts.AuthError{
			Message: fmt.Sprintf("Webex returned HTTP %d", resp.StatusCode),
			Hint:    "Check WEBEX_ACCESS_TOKEN or WEBEX_BEARER_TOKEN.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Webex returned HTTP %d", resp.StatusCode),
		}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.ValidationError{Message: "Webex returned a non-JSON response", Err: err}
	}
	return decoded, nil
}

var webexPathTemplateParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func resolveOperationPath(operation contracts.OperationDescriptor) (string, error) {
	path := operation.PathTemplate
	if strings.TrimSpace(path) == "" {
		path = operation.Path
	}
	if strings.TrimSpace(path) == "" {
		return "", contracts.ValidationError{Message: "operation path is required"}
	}

	missing := []string{}
	resolved := webexPathTemplateParamPattern.ReplaceAllStringFunc(path, func(token string) string {
		matches := webexPathTemplateParamPattern.FindStringSubmatch(token)
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
