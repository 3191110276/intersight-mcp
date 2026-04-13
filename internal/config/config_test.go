package config

import (
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/limits"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Endpoint != limits.DefaultEndpoint {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.OAuthTokenURL != "https://intersight.com/iam/token" {
		t.Fatalf("unexpected oauth token URL: %q", cfg.OAuthTokenURL)
	}
	if cfg.ProxyURL != "" {
		t.Fatalf("unexpected proxy URL: %q", cfg.ProxyURL)
	}
	if cfg.APIBaseURL != "https://intersight.com/api/v1" {
		t.Fatalf("unexpected API base URL: %q", cfg.APIBaseURL)
	}
	if cfg.Execution.GlobalTimeout != 40*time.Second {
		t.Fatalf("unexpected timeout: %v", cfg.Execution.GlobalTimeout)
	}
	if cfg.Execution.MaxOutputBytes != 512*1024 {
		t.Fatalf("unexpected max output: %d", cfg.Execution.MaxOutputBytes)
	}
	if cfg.LogLevel != LogLevelInfo {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
	}
	if cfg.UnsafeLogFullCode {
		t.Fatalf("expected log full code to default to false")
	}
	if cfg.LegacyContentMirror {
		t.Fatalf("expected legacy content mirror to default to false")
	}
	if cfg.SearchTimeout != limits.SearchTimeout || cfg.PerCallTimeout != limits.PerCallTimeout {
		t.Fatalf("unexpected hardcoded timeouts: %+v", cfg)
	}
	if cfg.MaxCodeSize != limits.MaxCodeSizeBytes {
		t.Fatalf("unexpected max code size: %d", cfg.MaxCodeSize)
	}
	if cfg.WASMMemory != limits.WASMMemoryBytes {
		t.Fatalf("unexpected wasm memory: %d", cfg.WASMMemory)
	}
}

func TestLoadConfigPrecedenceCLIOverEnv(t *testing.T) {
	t.Parallel()

	cfg, err := Load(
		[]string{
			"--endpoint", "https://flag.example.com",
			"--proxy", "http://proxy.flag.example.com:8080",
			"--timeout", "55s",
			"--max-output", "1MB",
			"--max-api-calls", "12",
			"--max-concurrent", "7",
			"--search-timeout", "22s",
			"--per-call-timeout", "11s",
			"--max-code-size", "256KB",
			"--wasm-memory", "96MB",
			"--log-level", "debug",
			"--read-only",
			"--unsafe-log-full-code",
			"--legacy-content-mirror",
		},
		[]string{
			"INTERSIGHT_CLIENT_ID=id",
			"INTERSIGHT_CLIENT_SECRET=secret",
			"INTERSIGHT_ENDPOINT=https://env.example.com",
			"INTERSIGHT_PROXY_URL=http://proxy.env.example.com:8080",
			"INTERSIGHT_TIMEOUT=15s",
			"INTERSIGHT_MAX_OUTPUT=2048",
			"INTERSIGHT_MAX_API_CALLS=9",
			"INTERSIGHT_MAX_CONCURRENT=3",
			"INTERSIGHT_SEARCH_TIMEOUT=8s",
			"INTERSIGHT_PER_CALL_TIMEOUT=4s",
			"INTERSIGHT_MAX_CODE_SIZE=32KB",
			"INTERSIGHT_WASM_MEMORY=32MB",
			"INTERSIGHT_LOG_LEVEL=info",
		},
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Endpoint != "https://flag.example.com" {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.Origin != "https://flag.example.com" {
		t.Fatalf("unexpected origin: %q", cfg.Origin)
	}
	if cfg.ProxyURL != "http://proxy.flag.example.com:8080" {
		t.Fatalf("unexpected proxy URL: %q", cfg.ProxyURL)
	}
	if cfg.Execution.GlobalTimeout != 55*time.Second {
		t.Fatalf("unexpected timeout: %v", cfg.Execution.GlobalTimeout)
	}
	if cfg.Execution.MaxOutputBytes != 1024*1024 {
		t.Fatalf("unexpected max output: %d", cfg.Execution.MaxOutputBytes)
	}
	if cfg.Execution.MaxAPICalls != 12 {
		t.Fatalf("unexpected max api calls: %d", cfg.Execution.MaxAPICalls)
	}
	if cfg.Execution.MaxConcurrent != 7 {
		t.Fatalf("unexpected max concurrent: %d", cfg.Execution.MaxConcurrent)
	}
	if cfg.SearchTimeout != 22*time.Second {
		t.Fatalf("unexpected search timeout: %v", cfg.SearchTimeout)
	}
	if cfg.PerCallTimeout != 11*time.Second {
		t.Fatalf("unexpected per-call timeout: %v", cfg.PerCallTimeout)
	}
	if cfg.MaxCodeSize != 256*1024 {
		t.Fatalf("unexpected max code size: %d", cfg.MaxCodeSize)
	}
	if cfg.WASMMemory != 96*1024*1024 {
		t.Fatalf("unexpected wasm memory: %d", cfg.WASMMemory)
	}
	if cfg.LogLevel != LogLevelDebug {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
	}
	if !cfg.ReadOnly {
		t.Fatalf("expected read-only mode to be enabled")
	}
	if !cfg.UnsafeLogFullCode {
		t.Fatalf("expected log full code to be enabled")
	}
	if !cfg.LegacyContentMirror {
		t.Fatalf("expected legacy content mirror to be enabled")
	}
}

func TestLoadConfigInvalidEndpoint(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, []string{
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

func TestLoadConfigInvalidProxy(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--proxy", "mailto://proxy"}, []string{
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

func TestLoadConfigRejectsHTTPEndpoint(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, []string{
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

func TestLoadConfigNormalizesEndpointToHTTPS(t *testing.T) {
	t.Parallel()

	cfg, err := Load(nil, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
		"INTERSIGHT_ENDPOINT=example.com:8443",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Endpoint != "https://example.com:8443" {
		t.Fatalf("unexpected endpoint: %q", cfg.Endpoint)
	}
	if cfg.Origin != "https://example.com:8443" {
		t.Fatalf("unexpected origin: %q", cfg.Origin)
	}
}

func TestLoadConfigInvalidMaxOutput(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--max-output", "abc"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid max-output error")
	}
	if !strings.Contains(err.Error(), "invalid max-output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigInvalidPositiveInts(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--max-api-calls", "0"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid max-api-calls error")
	}
	if !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigInvalidDurationKnobs(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--search-timeout", "0s"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid search-timeout error")
	}
	if !strings.Contains(err.Error(), "search-timeout") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = Load([]string{"--per-call-timeout", "nope"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid per-call-timeout error")
	}
	if !strings.Contains(err.Error(), "per-call-timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigInvalidSizeKnobs(t *testing.T) {
	t.Parallel()

	_, err := Load([]string{"--max-code-size", "abc"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid max-code-size error")
	}
	if !strings.Contains(err.Error(), "max-code-size") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = Load([]string{"--wasm-memory", "0"}, []string{
		"INTERSIGHT_CLIENT_ID=id",
		"INTERSIGHT_CLIENT_SECRET=secret",
	})
	if err == nil {
		t.Fatalf("expected invalid wasm-memory error")
	}
	if !strings.Contains(err.Error(), "wasm-memory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigMissingCredentialsAllowedForOfflineStartup(t *testing.T) {
	t.Parallel()

	cfg, err := Load(nil, nil)
	if err == nil {
		if cfg.HasCredentials() {
			t.Fatalf("expected credentials to be absent")
		}
		return
	}
	t.Fatalf("Load() error = %v", err)
}

func TestLoadConfigInvalidUnsafeLogFullCode(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, []string{
		"INTERSIGHT_UNSAFE_LOG_FULL_CODE=maybe",
	})
	if err == nil {
		t.Fatalf("expected invalid unsafe-log-full-code error")
	}
	if !strings.Contains(err.Error(), "invalid unsafe-log-full-code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigInvalidLegacyContentMirror(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, []string{
		"INTERSIGHT_LEGACY_CONTENT_MIRROR=maybe",
	})
	if err == nil {
		t.Fatalf("expected invalid legacy-content-mirror error")
	}
	if !strings.Contains(err.Error(), "invalid legacy-content-mirror") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServeFlagHelpMentionsSharedToolConcurrency(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var maxConcurrent int
	fs.IntVar(&maxConcurrent, "max-concurrent", 0, "maximum concurrent tool executions across search, query, and mutate")

	flagInfo := fs.Lookup("max-concurrent")
	if flagInfo == nil {
		t.Fatalf("expected max-concurrent flag to be registered")
	}
	if !strings.Contains(flagInfo.Usage, "search, query, and mutate") {
		t.Fatalf("unexpected usage text: %q", flagInfo.Usage)
	}
}
