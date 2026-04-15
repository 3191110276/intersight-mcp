package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	targetintersight "github.com/mimaurer/intersight-mcp/implementations/intersight"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestLoadArtifactBundleClonesInputsAndPreparesSharedState(t *testing.T) {
	artifacts := targetintersight.Artifacts()
	spec := append([]byte(nil), artifacts.ResolvedSpec...)
	catalog := append([]byte(nil), artifacts.SDKCatalog...)
	rules := append([]byte(nil), artifacts.Rules...)
	search := append([]byte(nil), artifacts.SearchCatalog...)

	bundle, err := LoadArtifactBundle(spec, catalog, rules, search)
	if err != nil {
		t.Fatalf("LoadArtifactBundle() error = %v", err)
	}

	spec[0] = 'x'
	catalog[0] = 'x'
	rules[0] = 'x'
	search[0] = 'x'

	if bundle.specIndex == nil {
		t.Fatal("bundle.specIndex is nil")
	}
	if bundle.sdk == nil {
		t.Fatal("bundle.sdk is nil")
	}
	searchExec, err := NewSearchExecutorFromBundle(testConfig(), bundle)
	if err != nil {
		t.Fatalf("NewSearchExecutorFromBundle() error = %v", err)
	}
	defer searchExec.Close()

	result, err := searchExec.Execute(context.Background(), `return Object.keys(catalog.resources || {}).length;`, ModeSearch)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	switch result.Value.(type) {
	case int64, float64:
	default:
		t.Fatalf("result.Value type = %T, want numeric", result.Value)
	}
}

func TestLoadArtifactBundleRejectsInvalidSearchJSON(t *testing.T) {
	artifacts := targetintersight.Artifacts()
	_, err := LoadArtifactBundle(
		artifacts.ResolvedSpec,
		artifacts.SDKCatalog,
		artifacts.Rules,
		[]byte("not-json"),
	)
	if err == nil {
		t.Fatal("expected error")
	}

	var validationErr contracts.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(validationErr.Error(), "search catalog") {
		t.Fatalf("unexpected error: %v", err)
	}
}
