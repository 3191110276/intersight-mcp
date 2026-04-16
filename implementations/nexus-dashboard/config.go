package nexusdashboard

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

const defaultDomain = "local"

type ConnectionConfig struct {
	Endpoint    string
	ProxyURLRaw string
	Username    string
	Password    string
	Domain      string
	APIKey      string
	Token       string
}

func (c ConnectionConfig) ProxyURL() string {
	return c.ProxyURLRaw
}

func (c ConnectionConfig) NewAPICaller(_ context.Context, _ time.Duration, httpClient *http.Client) implementations.APICaller {
	if !c.HasAuth() {
		return unavailableAPICaller{err: contracts.AuthError{Message: "Nexus Dashboard credentials are not configured; search is available, but query and mutate require a bearer token, username plus API key, or username plus password"}}
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
	var usernameFlag string
	var passwordFlag string
	var domainFlag string
	var apiKeyFlag string
	var tokenFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "base Nexus Dashboard endpoint origin")
	fs.StringVar(&proxyFlag, "proxy", "", "explicit proxy URL for outbound API traffic")
	fs.StringVar(&usernameFlag, "username", "", "Nexus Dashboard username")
	fs.StringVar(&passwordFlag, "password", "", "Nexus Dashboard password")
	fs.StringVar(&domainFlag, "domain", "", "Nexus Dashboard login domain")
	fs.StringVar(&apiKeyFlag, "api-key", "", "Nexus Dashboard API key")
	fs.StringVar(&tokenFlag, "token", "", "Nexus Dashboard bearer token or AuthCookie token")

	if err := fs.Parse(sharedconfig.FilterArgs(args, map[string]bool{
		"endpoint": true,
		"proxy":    true,
		"username": true,
		"password": true,
		"domain":   true,
		"api-key":  true,
		"token":    true,
	})); err != nil {
		return ConnectionConfig{}, err
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	cfg.Username = strings.TrimSpace(env["NEXUS_DASHBOARD_USERNAME"])
	cfg.Password = env["NEXUS_DASHBOARD_PASSWORD"]
	cfg.Domain = strings.TrimSpace(env["NEXUS_DASHBOARD_DOMAIN"])
	if cfg.Domain == "" {
		cfg.Domain = defaultDomain
	}
	cfg.APIKey = strings.TrimSpace(env["NEXUS_DASHBOARD_API_KEY"])
	cfg.Token = strings.TrimSpace(env["NEXUS_DASHBOARD_BEARER_TOKEN"])
	if cfg.Token == "" {
		cfg.Token = strings.TrimSpace(env["NEXUS_DASHBOARD_TOKEN"])
	}
	if cfg.Token == "" {
		cfg.Token = strings.TrimSpace(env["NEXUS_DASHBOARD_AUTH_COOKIE"])
	}

	if setFlags["username"] {
		cfg.Username = strings.TrimSpace(usernameFlag)
	}
	if setFlags["password"] {
		cfg.Password = passwordFlag
	}
	if setFlags["domain"] {
		cfg.Domain = strings.TrimSpace(domainFlag)
	}
	if strings.TrimSpace(cfg.Domain) == "" {
		cfg.Domain = defaultDomain
	}
	if setFlags["api-key"] {
		cfg.APIKey = strings.TrimSpace(apiKeyFlag)
	}
	if setFlags["token"] {
		cfg.Token = strings.TrimSpace(tokenFlag)
	}

	endpointRaw := env["NEXUS_DASHBOARD_ENDPOINT"]
	if setFlags["endpoint"] {
		endpointRaw = endpointFlag
	}
	parsedEndpoint, err := validateEndpoint(endpointRaw)
	if err != nil {
		return ConnectionConfig{}, err
	}
	cfg.Endpoint = parsedEndpoint.String()

	proxyRaw := env["NEXUS_DASHBOARD_PROXY_URL"]
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

func (c ConnectionConfig) HasAuth() bool {
	return c.HasToken() || c.HasAPIKey() || c.HasPasswordAuth()
}

func (c ConnectionConfig) HasToken() bool {
	return strings.TrimSpace(c.Token) != ""
}

func (c ConnectionConfig) HasAPIKey() bool {
	return strings.TrimSpace(c.Username) != "" && strings.TrimSpace(c.APIKey) != ""
}

func (c ConnectionConfig) HasPasswordAuth() bool {
	return strings.TrimSpace(c.Username) != "" && c.Password != ""
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
	if c.err != nil {
		return nil, c.err
	}
	return nil, contracts.AuthError{Message: "Nexus Dashboard credentials are not configured; search is available, but query and mutate require a bearer token, username plus API key, or username plus password"}
}
