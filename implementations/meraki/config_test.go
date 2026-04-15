package meraki

import "testing"

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
	if cfg.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestLoadConnectionConfigPrefersExplicitAPIKeyEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"MERAKI_API_KEY=preferred-key",
		"MERAKI_DASHBOARD_API_KEY=fallback-key",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.APIKey; got != "preferred-key" {
		t.Fatalf("APIKey = %q, want preferred-key", got)
	}
}

func TestLoadConnectionConfigFallsBackToOfficialDashboardAPIKeyEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"MERAKI_DASHBOARD_API_KEY=fallback-key",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.APIKey; got != "fallback-key" {
		t.Fatalf("APIKey = %q, want fallback-key", got)
	}
}

func TestLoadConnectionConfigFlagOverridesEndpointAndProxy(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig([]string{
		"--endpoint", "api.meraki.ca/api/v1",
		"--proxy", "http://proxy.example:8080",
	}, []string{
		"MERAKI_ENDPOINT=https://api.meraki.com/api/v1",
		"MERAKI_PROXY_URL=https://ignored.example:8443",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://api.meraki.ca/api/v1" {
		t.Fatalf("Endpoint = %q, want https://api.meraki.ca/api/v1", got)
	}
	if got := cfg.ProxyURLRaw; got != "http://proxy.example:8080" {
		t.Fatalf("ProxyURLRaw = %q, want http://proxy.example:8080", got)
	}
}

func TestLoadConnectionConfigNormalizesEndpointWithoutScheme(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"MERAKI_ENDPOINT=api.meraki.com",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://api.meraki.com/api/v1" {
		t.Fatalf("Endpoint = %q, want https://api.meraki.com/api/v1", got)
	}
}

func TestLoadConnectionConfigRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"MERAKI_ENDPOINT=http://api.meraki.com/api/v1",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
}

func TestLoadConnectionConfigRejectsInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"MERAKI_PROXY_URL=ftp://proxy.example",
	})
	if err == nil {
		t.Fatalf("expected invalid proxy error")
	}
}
