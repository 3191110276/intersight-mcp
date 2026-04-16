package catalystcenter

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

const defaultEndpoint = "https://sandboxdnac.cisco.com:443"

type ConnectionConfig struct {
	Endpoint    string
	ProxyURLRaw string
	Username    string
	Password    string
	StaticToken string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(_ context.Context, _ time.Duration, httpClient *http.Client) implementations.APICaller {
	if strings.TrimSpace(c.Endpoint) == "" {
		return unavailableAPICaller{err: contracts.AuthError{Message: "Catalyst Center endpoint is not configured; search is available, but query and mutate require CATALYST_CENTER_ENDPOINT"}}
	}
	if c.HasStaticToken() || c.HasCredentials() {
		return newAPICaller(c, httpClient)
	}
	return unavailableAPICaller{err: contracts.AuthError{Message: "Catalyst Center credentials are not configured; search is available, but query and mutate require either CATALYST_CENTER_X_AUTH_TOKEN or CATALYST_CENTER_USERNAME and CATALYST_CENTER_PASSWORD"}}
}

func LoadConnectionConfig(args []string, environ []string) (ConnectionConfig, error) {
	cfg := ConnectionConfig{}
	env := sharedconfig.ParseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var endpointFlag string
	var proxyFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "base Catalyst Center endpoint origin")
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

	cfg.Username = strings.TrimSpace(env["CATALYST_CENTER_USERNAME"])
	cfg.Password = strings.TrimSpace(env["CATALYST_CENTER_PASSWORD"])
	cfg.StaticToken = strings.TrimSpace(env["CATALYST_CENTER_X_AUTH_TOKEN"])

	endpointRaw := defaultEndpoint
	if value := strings.TrimSpace(env["CATALYST_CENTER_ENDPOINT"]); value != "" {
		endpointRaw = value
	}
	if setFlags["endpoint"] {
		endpointRaw = strings.TrimSpace(endpointFlag)
	}
	if strings.TrimSpace(endpointRaw) != "" {
		parsedEndpoint, err := validateEndpoint(endpointRaw)
		if err != nil {
			return ConnectionConfig{}, err
		}
		cfg.Endpoint = parsedEndpoint.String()
	}

	proxyRaw := env["CATALYST_CENTER_PROXY_URL"]
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
	return c.Username != "" && c.Password != ""
}

func (c ConnectionConfig) HasStaticToken() bool {
	return c.StaticToken != ""
}

func validateEndpoint(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("invalid endpoint %q: host is required", raw)
	}

	candidate := raw
	hasScheme := strings.Contains(candidate, "://")
	if !hasScheme {
		candidate = "https://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("invalid endpoint %q: must be an absolute URL", raw)
	}
	if hasScheme && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("invalid endpoint %q: scheme must be https when provided", raw)
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
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, fmt.Errorf("invalid endpoint %q: path is not allowed; use the origin only", raw)
	}

	parsed.Scheme = "https"
	parsed.Path = ""
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
