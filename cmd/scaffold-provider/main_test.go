package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreatesProviderScaffold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	err := run(runConfig{
		root:        root,
		provider:    "acme-api",
		withFilter:  true,
		withMetrics: true,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	expected := []string{
		"cmd/acme-api-mcp/main.go",
		"implementations/acme-api/implementation.go",
		"implementations/acme-api/config.go",
		"implementations/acme-api/filter.yaml",
		"implementations/acme-api/generated/embed.go",
		"implementations/acme-api/generated/spec_resolved.json",
		"implementations/acme-api/generated/sdk_catalog.json",
		"implementations/acme-api/generated/rules.json",
		"implementations/acme-api/generated/search_catalog.json",
		"third_party/acme-api/openapi/manifest.json",
		"third_party/acme-api/openapi/raw/.gitkeep",
		"third_party/acme-api/metrics/search_metrics.json",
	}
	for _, rel := range expected {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}

	implementationBytes, err := os.ReadFile(filepath.Join(root, "implementations/acme-api/implementation.go"))
	if err != nil {
		t.Fatalf("read implementation.go: %v", err)
	}
	implementation := string(implementationBytes)
	if !strings.Contains(implementation, `package acmeapi`) {
		t.Fatalf("implementation.go missing normalized package name: %s", implementation)
	}
	if !strings.Contains(implementation, `ServerName:      "acme-api-mcp"`) {
		t.Fatalf("implementation.go missing provider server name: %s", implementation)
	}
}

func TestRunRejectsExistingFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "cmd", "acme-mcp", "main.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := run(runConfig{root: root, provider: "acme"})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite existing file") {
		t.Fatalf("run() error = %v, want overwrite refusal", err)
	}
}
