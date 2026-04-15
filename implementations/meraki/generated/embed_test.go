package generated

import (
	"encoding/json"
	"testing"
)

func TestResolvedSpecBytes(t *testing.T) {
	t.Parallel()

	var got map[string]any
	if err := json.Unmarshal(ResolvedSpecBytes(), &got); err != nil {
		t.Fatalf("embedded resolved spec must be valid JSON: %v", err)
	}

	if len(got) == 0 {
		return
	}

	for _, key := range []string{"paths", "schemas", "tags"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("generated embedded spec missing %q", key)
		}
	}
	if len(got["paths"].(map[string]any)) == 0 {
		t.Fatalf("generated embedded spec must contain at least one path")
	}
	if len(got["schemas"].(map[string]any)) == 0 {
		t.Fatalf("generated embedded spec must contain at least one schema")
	}
}

func TestSDKCatalogBytes(t *testing.T) {
	t.Parallel()

	var got map[string]any
	if err := json.Unmarshal(SDKCatalogBytes(), &got); err != nil {
		t.Fatalf("embedded sdk catalog must be valid JSON: %v", err)
	}

	if len(got) == 0 {
		return
	}
	for _, key := range []string{"metadata", "methods"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("generated embedded sdk catalog missing %q", key)
		}
	}
}

func TestRulesBytes(t *testing.T) {
	t.Parallel()

	var got map[string]any
	if err := json.Unmarshal(RulesBytes(), &got); err != nil {
		t.Fatalf("embedded rules must be valid JSON: %v", err)
	}

	if len(got) == 0 {
		return
	}
	for _, key := range []string{"metadata", "methods"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("generated embedded rules missing %q", key)
		}
	}
}

func TestSearchCatalogBytes(t *testing.T) {
	t.Parallel()

	var got map[string]any
	if err := json.Unmarshal(SearchCatalogBytes(), &got); err != nil {
		t.Fatalf("embedded search catalog must be valid JSON: %v", err)
	}

	if len(got) == 0 {
		return
	}
	for _, key := range []string{"metadata", "resources", "resourceNames", "paths"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("generated embedded search catalog missing %q", key)
		}
	}
	if _, ok := got["methods"]; ok {
		t.Fatalf("generated embedded search catalog must not expose legacy %q", "methods")
	}
}
