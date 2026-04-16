package thousandeyes

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
	if cfg.Token != "" {
		t.Fatalf("Token = %q, want empty", cfg.Token)
	}
}

func TestLoadConnectionConfigPrefersExplicitBearerTokenEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"THOUSANDEYES_BEARER_TOKEN=preferred-token",
		"THOUSANDEYES_API_TOKEN=fallback-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Token; got != "preferred-token" {
		t.Fatalf("Token = %q, want preferred-token", got)
	}
}

func TestLoadConnectionConfigFallsBackToAPITokenEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"THOUSANDEYES_API_TOKEN=fallback-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Token; got != "fallback-token" {
		t.Fatalf("Token = %q, want fallback-token", got)
	}
}

func TestLoadConnectionConfigFlagOverridesEndpointAndProxy(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig([]string{
		"--endpoint", "api.thousandeyes.eu/v7",
		"--proxy", "http://proxy.example:8080",
	}, []string{
		"THOUSANDEYES_ENDPOINT=https://api.thousandeyes.com/v7",
		"THOUSANDEYES_PROXY_URL=https://ignored.example:8443",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://api.thousandeyes.eu/v7" {
		t.Fatalf("Endpoint = %q, want https://api.thousandeyes.eu/v7", got)
	}
	if got := cfg.ProxyURLRaw; got != "http://proxy.example:8080" {
		t.Fatalf("ProxyURLRaw = %q, want http://proxy.example:8080", got)
	}
}

func TestLoadConnectionConfigNormalizesEndpointWithoutScheme(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"THOUSANDEYES_ENDPOINT=api.thousandeyes.com",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://api.thousandeyes.com/v7" {
		t.Fatalf("Endpoint = %q, want https://api.thousandeyes.com/v7", got)
	}
}

func TestLoadConnectionConfigRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"THOUSANDEYES_ENDPOINT=http://api.thousandeyes.com/v7",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
}

func TestLoadConnectionConfigRejectsInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"THOUSANDEYES_PROXY_URL=ftp://proxy.example",
	})
	if err == nil {
		t.Fatalf("expected invalid proxy error")
	}
}
