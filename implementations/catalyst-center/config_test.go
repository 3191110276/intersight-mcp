package catalystcenter

import (
	"strings"
	"testing"
)

func TestLoadConnectionConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, nil)
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != defaultEndpoint {
		t.Fatalf("Endpoint = %q, want %q", got, defaultEndpoint)
	}
	if cfg.ProxyURLRaw != "" {
		t.Fatalf("ProxyURLRaw = %q, want empty", cfg.ProxyURLRaw)
	}
}

func TestLoadConnectionConfigPrefersStaticToken(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"CATALYST_CENTER_X_AUTH_TOKEN=token-123",
		"CATALYST_CENTER_USERNAME=user",
		"CATALYST_CENTER_PASSWORD=pass",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.StaticToken; got != "token-123" {
		t.Fatalf("StaticToken = %q, want token-123", got)
	}
}

func TestLoadConnectionConfigFlagOverridesEndpointAndProxy(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig([]string{
		"--endpoint", "dnac.example.com:8443",
		"--proxy", "http://proxy.example:8080",
	}, []string{
		"CATALYST_CENTER_ENDPOINT=https://ignored.example.com",
		"CATALYST_CENTER_PROXY_URL=https://ignored-proxy.example.com",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://dnac.example.com:8443" {
		t.Fatalf("Endpoint = %q, want https://dnac.example.com:8443", got)
	}
	if got := cfg.ProxyURLRaw; got != "http://proxy.example:8080" {
		t.Fatalf("ProxyURLRaw = %q, want http://proxy.example:8080", got)
	}
}

func TestLoadConnectionConfigRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"CATALYST_CENTER_ENDPOINT=https://example.com/path",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
	if !strings.Contains(err.Error(), "invalid endpoint") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConnectionConfigRejectsInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"CATALYST_CENTER_PROXY_URL=ftp://proxy.example",
	})
	if err == nil {
		t.Fatalf("expected invalid proxy error")
	}
	if !strings.Contains(err.Error(), "invalid proxy") {
		t.Fatalf("unexpected error: %v", err)
	}
}
