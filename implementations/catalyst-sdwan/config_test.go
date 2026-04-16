package catalystsdwan

import "testing"

func TestLoadConnectionConfigEndpointAndCredentials(t *testing.T) {
	cfg, err := LoadConnectionConfig(
		[]string{"--endpoint", "sdwan.example.com"},
		[]string{
			"CATALYST_SDWAN_USERNAME=admin",
			"CATALYST_SDWAN_PASSWORD=secret",
			"CATALYST_SDWAN_PROXY_URL=https://proxy.example.com",
		},
	)
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.Endpoint; got != "https://sdwan.example.com" {
		t.Fatalf("Endpoint = %q, want https://sdwan.example.com", got)
	}
	if got := cfg.ProxyURL(); got != "https://proxy.example.com" {
		t.Fatalf("ProxyURL = %q, want https://proxy.example.com", got)
	}
	if !cfg.HasCredentials() {
		t.Fatal("HasCredentials() = false, want true")
	}
}

func TestLoadConnectionConfigNormalizesSessionCookie(t *testing.T) {
	cfg, err := LoadConnectionConfig(nil, []string{
		"CATALYST_SDWAN_SESSION_COOKIE=session-token",
	})
	if err != nil {
		t.Fatalf("LoadConnectionConfig() error = %v", err)
	}
	if got := cfg.SessionCookie(); got != "JSESSIONID=session-token" {
		t.Fatalf("SessionCookie() = %q, want JSESSIONID=session-token", got)
	}
}

func TestValidateEndpointRejectsDataserviceQuery(t *testing.T) {
	_, err := validateEndpoint("https://sdwan.example.com/dataservice?x=1")
	if err == nil {
		t.Fatal("validateEndpoint() error = nil, want error")
	}
}
