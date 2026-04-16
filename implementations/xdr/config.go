package xdr

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mimaurer/intersight-mcp/implementations"
	sharedconfig "github.com/mimaurer/intersight-mcp/internal/config"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	ciscointersight "github.com/mimaurer/intersight-mcp/intersight"
)

const (
	defaultEndpoint      = "https://automate.us.security.cisco.com/api"
	defaultOAuthTokenURL = "https://visibility.amp.cisco.com/iroh/oauth2/token"
	defaultValidatePath  = "/v1/workflows/summary"
)

type ConnectionConfig struct {
	Endpoint         string
	ProxyURLRaw      string
	AccessToken      string
	ClientID         string
	ClientSecret     string
	OAuthTokenURL    string
	OAuthValidateURL string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(ctx context.Context, _ time.Duration, httpClient *http.Client) implementations.APICaller {
	switch {
	case strings.TrimSpace(c.AccessToken) != "":
		return ciscointersight.NewClient(httpClient, c.Endpoint, staticTokenProvider{token: c.AccessToken})
	case c.HasCredentials():
		manager, err := ciscointersight.NewOAuthManager(ctx, ciscointersight.OAuthConfig{
			TokenURL:     c.OAuthTokenURL,
			ValidateURL:  c.OAuthValidateURL,
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			HTTPClient:   httpClient,
		})
		if err != nil {
			return unavailableAPICaller{err: err}
		}
		return ciscointersight.NewClient(httpClient, c.Endpoint, manager)
	default:
		return unavailableAPICaller{err: contracts.AuthError{Message: "Cisco XDR credentials are not configured; search is available, but query and mutate require XDR_ACCESS_TOKEN or XDR_CLIENT_ID and XDR_CLIENT_SECRET"}}
	}
}

func LoadConnectionConfig(args []string, environ []string) (ConnectionConfig, error) {
	cfg := ConnectionConfig{}
	env := sharedconfig.ParseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var endpointFlag string
	var proxyFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "base Cisco XDR Automation API endpoint")
	fs.StringVar(&proxyFlag, "proxy", "", "explicit proxy URL for outbound OAuth and API traffic")

	if err := fs.Parse(sharedconfig.FilterArgs(args, map[string]bool{
		"endpoint": true,
		"proxy":    true,
	})); err != nil {
		return ConnectionConfig{}, err
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	cfg.AccessToken = strings.TrimSpace(env["XDR_ACCESS_TOKEN"])
	if cfg.AccessToken == "" {
		cfg.AccessToken = strings.TrimSpace(env["XDR_API_TOKEN"])
	}
	cfg.ClientID = strings.TrimSpace(env["XDR_CLIENT_ID"])
	cfg.ClientSecret = strings.TrimSpace(env["XDR_CLIENT_SECRET"])
	cfg.OAuthTokenURL = defaultOAuthTokenURL

	endpointRaw := defaultEndpoint
	if value := strings.TrimSpace(env["XDR_ENDPOINT"]); value != "" {
		endpointRaw = value
	}
	if setFlags["endpoint"] {
		endpointRaw = strings.TrimSpace(endpointFlag)
	}

	parsedEndpoint, err := validateEndpoint(endpointRaw)
	if err != nil {
		return ConnectionConfig{}, err
	}
	cfg.Endpoint = parsedEndpoint.String()
	cfg.OAuthValidateURL = strings.TrimRight(cfg.Endpoint, "/") + defaultValidatePath

	proxyRaw := env["XDR_PROXY_URL"]
	if setFlags["proxy"] {
		proxyRaw = proxyFlag
	}
	if strings.TrimSpace(proxyRaw) != "" {
		parsedProxy, err := validateProxyURL(proxyRaw)
		if err != nil {
			return ConnectionConfig{}, err
		}
		cfg.ProxyURLRaw = parsedProxy.String()
	}

	return cfg, nil
}

func (c ConnectionConfig) HasCredentials() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

func validateEndpoint(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("invalid endpoint %q: URL is required", raw)
	}

	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("invalid endpoint %q: must be an absolute URL", raw)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("invalid endpoint %q: scheme must be https", raw)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid endpoint %q: host is required", raw)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("invalid endpoint %q: user info is not allowed", raw)
	}
	if parsed.RawQuery != "" {
		return nil, fmt.Errorf("invalid endpoint %q: query is not allowed", raw)
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid endpoint %q: fragment is not allowed", raw)
	}

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		path = "/api"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parsed.Scheme = "https"
	parsed.Path = path
	parsed.RawPath = ""
	return parsed, nil
}

func validateProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("invalid proxy %q: URL is required", raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("invalid proxy %q: must be an absolute URL", raw)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid proxy %q: host is required", raw)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("invalid proxy %q: scheme must be http, https, or socks5", raw)
	}
	if parsed.RawQuery != "" {
		return nil, fmt.Errorf("invalid proxy %q: query is not allowed", raw)
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid proxy %q: fragment is not allowed", raw)
	}
	return parsed, nil
}

type unavailableAPICaller struct {
	err error
}

func (c unavailableAPICaller) Do(_ context.Context, _ contracts.OperationDescriptor) (any, error) {
	return nil, c.err
}

type staticTokenProvider struct {
	token string
}

func (p staticTokenProvider) Token(_ context.Context) (string, error) {
	if strings.TrimSpace(p.token) == "" {
		return "", contracts.AuthError{Message: "Cisco XDR access token is not configured"}
	}
	return p.token, nil
}
