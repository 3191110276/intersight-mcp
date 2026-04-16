package nexusdashboard

import "testing"

func TestLoadConnectionConfigUsesTokenAndNormalizesEndpoint(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"NEXUS_DASHBOARD_ENDPOINT=nd.example.com",
		"NEXUS_DASHBOARD_BEARER_TOKEN=test-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://nd.example.com" {
		t.Fatalf("Endpoint = %q, want https://nd.example.com", got)
	}
	if got := cfg.Token; got != "test-token" {
		t.Fatalf("Token = %q, want test-token", got)
	}
	if !cfg.HasToken() {
		t.Fatalf("HasToken() = false, want true")
	}
}

func TestLoadConnectionConfigDefaultsDomainForPasswordAuth(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConnectionConfig(nil, []string{
		"NEXUS_DASHBOARD_ENDPOINT=https://nd.example.com",
		"NEXUS_DASHBOARD_USERNAME=admin",
		"NEXUS_DASHBOARD_PASSWORD=secret",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Domain; got != defaultDomain {
		t.Fatalf("Domain = %q, want %q", got, defaultDomain)
	}
	if !cfg.HasPasswordAuth() {
		t.Fatalf("HasPasswordAuth() = false, want true")
	}
}

func TestLoadConnectionConfigRejectsEndpointPath(t *testing.T) {
	t.Parallel()

	_, err := LoadConnectionConfig(nil, []string{
		"NEXUS_DASHBOARD_ENDPOINT=https://nd.example.com/api/v1/manage",
	})
	if err == nil {
		t.Fatalf("LoadConnectionConfig() error = nil, want invalid endpoint error")
	}
}
