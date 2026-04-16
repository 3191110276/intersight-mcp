package xdr

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
	if got := cfg.OAuthTokenURL; got != defaultOAuthTokenURL {
		t.Fatalf("OAuthTokenURL = %q, want %q", got, defaultOAuthTokenURL)
	}
	if got := cfg.OAuthValidateURL; got != defaultEndpoint+defaultValidatePath {
		t.Fatalf("OAuthValidateURL = %q, want %q", got, defaultEndpoint+defaultValidatePath)
	}
}

func TestLoadConnectionConfigUsesAccessTokenEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"XDR_ACCESS_TOKEN=token-123",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.AccessToken; got != "token-123" {
		t.Fatalf("AccessToken = %q, want token-123", got)
	}
}

func TestLoadConnectionConfigUsesClientCredentialsEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"XDR_CLIENT_ID=client-id",
		"XDR_CLIENT_SECRET=client-secret",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if !cfg.HasCredentials() {
		t.Fatalf("expected HasCredentials() to be true")
	}
}

func TestLoadConnectionConfigFlagOverridesEndpointAndProxy(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig([]string{
		"--endpoint", "xdr.eu.security.cisco.com/api",
		"--proxy", "http://proxy.example:8080",
	}, []string{
		"XDR_ENDPOINT=https://automate.us.security.cisco.com/api",
		"XDR_PROXY_URL=https://ignored.example:8443",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://xdr.eu.security.cisco.com/api" {
		t.Fatalf("Endpoint = %q, want https://xdr.eu.security.cisco.com/api", got)
	}
	if got := cfg.ProxyURLRaw; got != "http://proxy.example:8080" {
		t.Fatalf("ProxyURLRaw = %q, want http://proxy.example:8080", got)
	}
}

func TestLoadConnectionConfigRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"XDR_ENDPOINT=http://automate.us.security.cisco.com/api",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint error")
	}
}

func TestLoadConnectionConfigRejectsInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"XDR_PROXY_URL=ftp://proxy.example",
	})
	if err == nil {
		t.Fatalf("expected invalid proxy error")
	}
}
