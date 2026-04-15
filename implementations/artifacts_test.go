package implementations_test

import (
	"testing"

	"github.com/mimaurer/intersight-mcp/implementations"
)

func TestStandardGenerationConfig(t *testing.T) {
	t.Parallel()

	cfg := implementations.StandardGenerationConfig("acme", implementations.StandardGenerationConfigOptions{
		IncludeFilter:        true,
		IncludeMetrics:       true,
		FallbackPathPrefixes: []string{"/v1/"},
	})

	if got := cfg.RawSpecPath; got != "third_party/acme/openapi/raw/openapi.json" {
		t.Fatalf("RawSpecPath = %q", got)
	}
	if got := cfg.ManifestPath; got != "third_party/acme/openapi/manifest.json" {
		t.Fatalf("ManifestPath = %q", got)
	}
	if got := cfg.FilterPath; got != "implementations/acme/filter.yaml" {
		t.Fatalf("FilterPath = %q", got)
	}
	if got := cfg.MetricsPath; got != "third_party/acme/metrics/search_metrics.json" {
		t.Fatalf("MetricsPath = %q", got)
	}
	if got := cfg.OutputPath; got != "implementations/acme/generated/spec_resolved.json" {
		t.Fatalf("OutputPath = %q", got)
	}
	if len(cfg.FallbackPathPrefixes) != 1 || cfg.FallbackPathPrefixes[0] != "/v1/" {
		t.Fatalf("FallbackPathPrefixes = %#v", cfg.FallbackPathPrefixes)
	}
}
