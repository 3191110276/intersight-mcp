package sandbox

import (
	"encoding/json"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

// ArtifactBundle holds immutable, pre-parsed startup artifacts that can be
// shared safely across executor instances.
type ArtifactBundle struct {
	specIndex *dryRunSpecIndex
	sdk       *sdkRuntime
	search    *searchRuntime
}

// LoadArtifactBundle parses and prepares the embedded artifacts once so later
// executor construction can reuse the same immutable state.
func LoadArtifactBundle(specJSON, catalogJSON, rulesJSON, searchJSON []byte) (*ArtifactBundle, error) {
	return LoadArtifactBundleWithExtensions(specJSON, catalogJSON, rulesJSON, searchJSON, Extensions{})
}

func LoadArtifactBundleWithExtensions(specJSON, catalogJSON, rulesJSON, searchJSON []byte, ext Extensions) (*ArtifactBundle, error) {
	if !json.Valid(specJSON) {
		return nil, contracts.ValidationError{Message: "embedded spec is not valid JSON"}
	}
	if !json.Valid(catalogJSON) {
		return nil, contracts.ValidationError{Message: "embedded sdk catalog is not valid JSON"}
	}
	if !json.Valid(rulesJSON) {
		return nil, contracts.ValidationError{Message: "embedded rules are not valid JSON"}
	}
	if !json.Valid(searchJSON) {
		return nil, contracts.ValidationError{Message: "embedded search catalog is not valid JSON"}
	}

	specIndex, err := loadDryRunSpecIndex(specJSON, ext)
	if err != nil {
		return nil, err
	}
	sdk, err := loadSDKRuntime(specJSON, catalogJSON, rulesJSON, ext)
	if err != nil {
		return nil, err
	}
	publicSearchJSON, err := redactSearchCatalogPublicFields(searchJSON)
	if err != nil {
		return nil, err
	}
	var searchCatalog contracts.SearchCatalog
	if err := json.Unmarshal(publicSearchJSON, &searchCatalog); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded search catalog", Err: err}
	}
	search := newSearchRuntime(searchCatalog, sdk.spec.Schemas)

	return &ArtifactBundle{
		specIndex: specIndex,
		sdk:       sdk,
		search:    search,
	}, nil
}
