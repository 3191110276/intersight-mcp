package config

import (
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
	if cfg.SearchTimeout != limits.SearchTimeout || cfg.PerCallTimeout != limits.PerCallTimeout {
		t.Fatalf("unexpected hardcoded timeouts: %+v", cfg)
	}
}

func TestLoadConfigPrecedenceCLIOverEnv(t *testing.T) {
	t.Parallel()

	cfg, err := Load(
		[]string{
			"--endpoint", "https://flag.example.com",
			"--timeout", "55s",
			"--max-output", "1MB",
			"--max-api-calls", "12",
			"--max-concurrent", "7",
			"--log-level", "debug",
		},
		[]string{
			"INTERSIGHT_CLIENT_ID=id",
			"INTERSIGHT_CLIENT_SECRET=secret",
			"INTERSIGHT_ENDPOINT=https://env.example.com",
			"INTERSIGHT_TIMEOUT=15s",
			"INTERSIGHT_MAX_OUTPUT=2048",
			"INTERSIGHT_MAX_API_CALLS=9",
			"INTERSIGHT_MAX_CONCURRENT=3",
			"INTERSIGHT_LOG_LEVEL=info",
		},
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Origin != "https://flag.example.com" {
		t.Fatalf("unexpected origin: %q", cfg.Origin)
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
	if cfg.LogLevel != LogLevelDebug {
		t.Fatalf("unexpected log level: %q", cfg.LogLevel)
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

func TestLoadConfigMissingCredentials(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, nil)
	if err == nil {
		t.Fatalf("expected missing credentials error")
	}
	if !strings.Contains(err.Error(), "INTERSIGHT_CLIENT_ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}
