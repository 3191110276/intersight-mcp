package intersight

import (
	"strings"
	"testing"
)

func TestLoadConnectionConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}

	if cfg.Endpoint != defaultEndpoint {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.OAuthTokenURL != "https://intersight.com/iam/token" {
		t.Fatalf("unexpected oauth token URL: %q", cfg.OAuthTokenURL)
	}
	if cfg.ProxyURL() != "" {
		t.Fatalf("unexpected proxy URL: %q", cfg.ProxyURL())
	}
	if cfg.APIBaseURL != "https://intersight.com/api/v1" {
		t.Fatalf("unexpected API base URL: %q", cfg.APIBaseURL)
	}
}

func TestLoadConnectionConfigPrecedenceCLIOverEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(
		[]string{
			"--endpoint", "https://flag.example.com",
			"--proxy", "http://proxy.flag.example.com:8080",
		},
		[]string{
			"INTERSIGHT_CLIENT_ID=id",
			"INTERSIGHT_CLIENT_SECRET=secret",
			"INTERSIGHT_ENDPOINT=https://env.example.com",
			"INTERSIGHT_PROXY_URL=http://proxy.env.example.com:8080",
		},
	)
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}

	if cfg.Endpoint != "https://flag.example.com" {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.Origin != "https://flag.example.com" {
		t.Fatalf("unexpected origin: %q", cfg.Origin)
	}
	if cfg.ProxyURL() != "http://proxy.flag.example.com:8080" {
		t.Fatalf("unexpected proxy URL: %q", cfg.ProxyURL())
	}
}

func TestLoadConnectionConfigInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=https://example.com/path?x=1",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
	if !strings.Contains(err.Error(), "invalid endpoint") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConnectionConfigInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig([]string{"--proxy", "mailto://proxy"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid proxy error")
	}
	if !strings.Contains(err.Error(), "invalid proxy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConnectionConfigRejectsHTTPEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=http://example.com",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
	if !strings.Contains(err.Error(), "scheme must be https") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConnectionConfigNormalizesEndpointToHTTPS(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=example.com:8443",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}

	if cfg.Endpoint != "https://example.com:8443" {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.Origin != "https://example.com:8443" {
		t.Fatalf("unexpected origin: %q", cfg.Origin)
	}
}

func TestLoadConnectionConfigMissingCredentialsAllowedForOfflineStartup(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, nil)
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if cfg.HasCredentials() {
		t.Fatalf("expected credentials to be absent")
	}
}
