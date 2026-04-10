package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mimaurer/intersight-mcp/generated"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestLoadArtifactBundleClonesInputsAndPreparesSharedState(t *testing.T) {
	spec := append([]byte(nil), generated.ResolvedSpecBytes()...)
	catalog := append([]byte(nil), generated.SDKCatalogBytes()...)
	rules := append([]byte(nil), generated.RulesBytes()...)
	search := append([]byte(nil), generated.SearchCatalogBytes()...)

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
	if !json.Valid(bundle.publicSearchJSON) {
		t.Fatal("bundle.publicSearchJSON is not valid JSON")
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
	_, err := LoadArtifactBundle(
		generated.ResolvedSpecBytes(),
		generated.SDKCatalogBytes(),
		generated.RulesBytes(),
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
