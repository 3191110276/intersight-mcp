package catalystcenter

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
	"sync"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	internalpkg "github.com/mimaurer/intersight-mcp/internal"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const maxCatalystCenterResponseBytes = 16 << 20
const authTokenPath = "/dna/system/api/v1/auth/token"

type Client struct {
	httpClient *http.Client
	cfg        ConnectionConfig

	mu    sync.Mutex
	token string
}

func NewClient(httpClient *http.Client, cfg ConnectionConfig) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		httpClient: httpClient,
		cfg:        cfg,
		token:      strings.TrimSpace(cfg.StaticToken),
	}
}

func newAPICaller(cfg ConnectionConfig, httpClient *http.Client) implementations.APICaller {
	return NewClient(httpClient, cfg)
}

func (c *Client) Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, err
	}

	requestPath, err := resolveOperationPath(operation)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(operation.EndpointURL)
	if endpoint == "" {
		endpoint = joinEndpoint(c.cfg.Endpoint, requestPath)
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
		return nil, contracts.InternalError{Message: "build Catalyst Center request", Err: err}
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
	req.Header.Set("X-Auth-Token", token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		internalpkg.RecordAPICall(ctx, internalpkg.APICallRecord{
			Method:     operation.Method,
			Path:       req.URL.Path,
			DurationMS: time.Since(start).Milliseconds(),
		})
		return nil, contracts.NetworkError{Message: "Catalyst Center request failed", Err: err}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxCatalystCenterResponseBytes+1))
	if readErr != nil {
		return nil, contracts.NetworkError{Message: "read Catalyst Center response failed", Err: readErr}
	}
	if len(body) > maxCatalystCenterResponseBytes {
		return nil, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Catalyst Center response exceeded the %d MiB limit", maxCatalystCenterResponseBytes/(1<<20)),
		}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, contracts.AuthError{
			Message: fmt.Sprintf("Catalyst Center returned HTTP %d", resp.StatusCode),
			Hint:    "Check CATALYST_CENTER_X_AUTH_TOKEN.",
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, contracts.HTTPError{
			Status:  resp.StatusCode,
			Body:    decodeBody(body),
			Message: fmt.Sprintf("Catalyst Center returned HTTP %d", resp.StatusCode),
		}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, contracts.ValidationError{Message: "Catalyst Center returned a non-JSON response", Err: err}
	}
	return decoded, nil
}

func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if strings.TrimSpace(c.token) != "" {
		return c.token, nil
	}
	if !c.cfg.HasCredentials() {
		return "", contracts.AuthError{Message: "Catalyst Center credentials are not configured; query and mutate require CATALYST_CENTER_X_AUTH_TOKEN or CATALYST_CENTER_USERNAME and CATALYST_CENTER_PASSWORD"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinEndpoint(c.cfg.Endpoint, authTokenPath), nil)
	if err != nil {
		return "", contracts.InternalError{Message: "build Catalyst Center auth request", Err: err}
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", contracts.NetworkError{Message: "Catalyst Center token request failed", Err: err}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return "", contracts.NetworkError{Message: "read Catalyst Center token response failed", Err: readErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", contracts.AuthError{
			Message: fmt.Sprintf("Catalyst Center token request failed with HTTP %d", resp.StatusCode),
			Err:     contracts.HTTPError{Status: resp.StatusCode, Body: decodeBody(body)},
		}
	}

	var payload struct {
		Token string `json:"Token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", contracts.AuthError{Message: "Catalyst Center token response was not valid JSON", Err: err}
	}
	if strings.TrimSpace(payload.Token) == "" {
		return "", contracts.AuthError{Message: "Catalyst Center token response did not include Token"}
	}
	c.token = payload.Token
	return c.token, nil
}

var catalystCenterPathTemplateParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func resolveOperationPath(operation contracts.OperationDescriptor) (string, error) {
	path := operation.PathTemplate
	if strings.TrimSpace(path) == "" {
		path = operation.Path
	}
	if strings.TrimSpace(path) == "" {
		return "", contracts.ValidationError{Message: "operation path is required"}
	}

	missing := []string{}
	resolved := catalystCenterPathTemplateParamPattern.ReplaceAllStringFunc(path, func(token string) string {
		matches := catalystCenterPathTemplateParamPattern.FindStringSubmatch(token)
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
