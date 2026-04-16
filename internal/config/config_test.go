package config

import (
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/limits"
)

func TestLoadRuntimeDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadRuntime(nil, nil, "INTERSIGHT")
	if err != nil {
		t.Fatalf("LoadRuntime() error = %v", err)
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

func TestLoadRuntimePrecedenceCLIOverEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadRuntime(
		[]string{
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
			"INTERSIGHT_TIMEOUT=15s",
			"INTERSIGHT_MAX_OUTPUT=2048",
			"INTERSIGHT_MAX_API_CALLS=9",
			"INTERSIGHT_MAX_CONCURRENT=3",
			"INTERSIGHT_SEARCH_TIMEOUT=8s",
			"INTERSIGHT_PER_CALL_TIMEOUT=4s",
			"INTERSIGHT_MAX_CODE_SIZE=32KB",
			"INTERSIGHT_WASM_MEMORY=32MB",
			"INTERSIGHT_LOG_LEVEL=info",
			"INTERSIGHT_READ_ONLY=false",
		},
		"INTERSIGHT",
	)
	if err != nil {
		t.Fatalf("LoadRuntime() error = %v", err)
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

func TestLoadRuntimeReadOnlyFromEnv(t *testing.T) {
	t.Parallel()

	cfg, err := LoadRuntime(nil, []string{"INTERSIGHT_READ_ONLY=true"}, "INTERSIGHT")
	if err != nil {
		t.Fatalf("LoadRuntime() error = %v", err)
	}

	if !cfg.ReadOnly {
		t.Fatalf("expected read-only mode to be enabled from env")
	}
}

func TestLoadRuntimeInvalidReadOnly(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime(nil, []string{"INTERSIGHT_READ_ONLY=maybe"}, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid read-only error")
	}
	if !strings.Contains(err.Error(), "invalid read-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestLoadRuntimeInvalidMaxOutput(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime([]string{"--max-output", "abc"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid max-output error")
	}
	if !strings.Contains(err.Error(), "invalid max-output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRuntimeInvalidPositiveInts(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime([]string{"--max-api-calls", "0"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid max-api-calls error")
	}
	if !strings.Contains(err.Error(), "positive integer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRuntimeInvalidDurationKnobs(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime([]string{"--search-timeout", "0s"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid search-timeout error")
	}
	if !strings.Contains(err.Error(), "search-timeout") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = LoadRuntime([]string{"--per-call-timeout", "nope"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid per-call-timeout error")
	}
	if !strings.Contains(err.Error(), "per-call-timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRuntimeInvalidSizeKnobs(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime([]string{"--max-code-size", "abc"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid max-code-size error")
	}
	if !strings.Contains(err.Error(), "max-code-size") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = LoadRuntime([]string{"--wasm-memory", "0"}, nil, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid wasm-memory error")
	}
	if !strings.Contains(err.Error(), "wasm-memory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRuntimeInvalidUnsafeLogFullCode(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime(nil, []string{"INTERSIGHT_UNSAFE_LOG_FULL_CODE=maybe"}, "INTERSIGHT")
	if err == nil {
		t.Fatalf("expected invalid unsafe-log-full-code error")
	}
	if !strings.Contains(err.Error(), "invalid unsafe-log-full-code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRuntimeInvalidLegacyContentMirror(t *testing.T) {
	t.Parallel()

	_, err := LoadRuntime(nil, []string{"INTERSIGHT_LEGACY_CONTENT_MIRROR=maybe"}, "INTERSIGHT")
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
