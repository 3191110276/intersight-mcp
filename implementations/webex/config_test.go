package webex

import "testing"

func TestLoadConnectionConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"WEBEX_ACCESS_TOKEN=test-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if cfg.Endpoint != defaultEndpoint {
		t.Fatalf("Endpoint = %q, want %q", cfg.Endpoint, defaultEndpoint)
	}
	if cfg.AccessToken != "test-token" {
		t.Fatalf("AccessToken = %q, want test-token", cfg.AccessToken)
	}
}

func TestLoadConnectionConfigPrefersBearerTokenFallbacks(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"WEBEX_BEARER_TOKEN=bearer-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if cfg.AccessToken != "bearer-token" {
		t.Fatalf("AccessToken = %q, want bearer-token", cfg.AccessToken)
	}
}

func TestLoadConnectionConfigRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig([]string{"--endpoint", "http://webexapis.com/v1"}, nil)
	if err == nil {
		t.Fatal("LoadConnectionConfig() error = nil, want invalid endpoint")
	}
}
