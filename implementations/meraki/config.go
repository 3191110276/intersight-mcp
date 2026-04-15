package meraki

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

const defaultEndpoint = "https://api.meraki.com/api/v1"

type ConnectionConfig struct {
	Endpoint    string
	ProxyURLRaw string
	APIKey      string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(_ context.Context, _ time.Duration, httpClient *http.Client) implementations.APICaller {
	if !c.HasAPIKey() {
		return unavailableClient{err: contracts.AuthError{Message: "Meraki API key is not configured; search is available, but query and mutate require MERAKI_API_KEY or MERAKI_DASHBOARD_API_KEY"}}
	}
	return newAPICaller(c, httpClient)
}

func LoadConnectionConfig(args []string, environ []string) (ConnectionConfig, error) {
	cfg := ConnectionConfig{}
	env := sharedconfig.ParseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var endpointFlag string
	var proxyFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "base Meraki API endpoint")
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

	cfg.APIKey = strings.TrimSpace(env["MERAKI_API_KEY"])
	if cfg.APIKey == "" {
		cfg.APIKey = strings.TrimSpace(env["MERAKI_DASHBOARD_API_KEY"])
	}

	endpointRaw := defaultEndpoint
	if value := strings.TrimSpace(env["MERAKI_ENDPOINT"]); value != "" {
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

	proxyRaw := env["MERAKI_PROXY_URL"]
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

func (c ConnectionConfig) HasAPIKey() bool {
	return c.APIKey != ""
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
		path = "/api/v1"
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
