package catalystsdwan

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
)

type ConnectionConfig struct {
	Endpoint         string
	ProxyURLRaw      string
	Username         string
	Password         string
	BearerToken      string
	SessionCookieRaw string
	XSRFToken        string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(_ context.Context, _ time.Duration, httpClient *http.Client) implementations.APICaller {
	if strings.TrimSpace(c.Endpoint) == "" {
		return unavailableClient{err: contracts.AuthError{Message: "Catalyst SD-WAN endpoint is not configured; search is available, but query and mutate require CATALYST_SDWAN_ENDPOINT"}}
	}
	if !c.HasCredentials() {
		return unavailableClient{err: contracts.AuthError{Message: "Catalyst SD-WAN credentials are not configured; set username/password, bearer token, or session cookie"}}
	}
	return NewClient(httpClient, c)
}

func LoadConnectionConfig(args []string, environ []string) (ConnectionConfig, error) {
	cfg := ConnectionConfig{}
	env := sharedconfig.ParseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var endpointFlag string
	var proxyFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "Catalyst SD-WAN Manager origin")
	fs.StringVar(&proxyFlag, "proxy", "", "explicit proxy URL for outbound API traffic")

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

	cfg.Username = strings.TrimSpace(env["CATALYST_SDWAN_USERNAME"])
	cfg.Password = env["CATALYST_SDWAN_PASSWORD"]
	cfg.BearerToken = strings.TrimSpace(env["CATALYST_SDWAN_BEARER_TOKEN"])
	cfg.SessionCookieRaw = strings.TrimSpace(env["CATALYST_SDWAN_SESSION_COOKIE"])
	cfg.XSRFToken = strings.TrimSpace(env["CATALYST_SDWAN_XSRF_TOKEN"])

	endpointRaw := strings.TrimSpace(env["CATALYST_SDWAN_ENDPOINT"])
	if setFlags["endpoint"] {
		endpointRaw = strings.TrimSpace(endpointFlag)
	}
	if endpointRaw != "" {
		parsedEndpoint, err := validateEndpoint(endpointRaw)
		if err != nil {
			return ConnectionConfig{}, err
		}
		cfg.Endpoint = parsedEndpoint.String()
	}

	proxyRaw := env["CATALYST_SDWAN_PROXY_URL"]
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
	if c.Username != "" && c.Password != "" {
		return true
	}
	if c.BearerToken != "" {
		return true
	}
	return c.SessionCookie() != ""
}

func (c ConnectionConfig) SessionCookie() string {
	raw := strings.TrimSpace(c.SessionCookieRaw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "=") {
		return raw
	}
	return "JSESSIONID=" + raw
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
	if path == "/dataservice" {
		path = ""
	}
	if path != "" && !strings.HasPrefix(path, "/") {
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

type unavailableClient struct {
	err error
}

func (c unavailableClient) Do(_ context.Context, _ contracts.OperationDescriptor) (any, error) {
	return nil, c.err
}
