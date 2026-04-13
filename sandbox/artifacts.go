package sandbox

import (
	"encoding/json"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

// ArtifactBundle holds immutable, pre-parsed startup artifacts that can be
// shared safely across executor instances.
type ArtifactBundle struct {
	specJSON         []byte
	catalogJSON      []byte
	rulesJSON        []byte
	searchJSON       []byte
	publicSearchJSON []byte
	specIndex        *dryRunSpecIndex
	sdk              *sdkRuntime
	search           *searchRuntime
}

// LoadArtifactBundle parses and prepares the embedded artifacts once so later
// executor construction can reuse the same immutable state.
func LoadArtifactBundle(specJSON, catalogJSON, rulesJSON, searchJSON []byte) (*ArtifactBundle, error) {
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

	specCopy := append([]byte(nil), specJSON...)
	catalogCopy := append([]byte(nil), catalogJSON...)
	rulesCopy := append([]byte(nil), rulesJSON...)
	searchCopy := append([]byte(nil), searchJSON...)

	specIndex, err := loadDryRunSpecIndex(specCopy)
	if err != nil {
		return nil, err
	}
	sdk, err := loadSDKRuntime(specCopy, catalogCopy, rulesCopy)
	if err != nil {
		return nil, err
	}
	publicSearchJSON, err := redactSearchCatalogPublicFields(searchCopy)
	if err != nil {
		return nil, err
	}
	var searchCatalog contracts.SearchCatalog
	if err := json.Unmarshal(publicSearchJSON, &searchCatalog); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded search catalog", Err: err}
	}
	search := newSearchRuntime(sdk.spec, sdk.catalog, sdk.rules, searchCatalog)

	return &ArtifactBundle{
		specJSON:         specCopy,
		catalogJSON:      catalogCopy,
		rulesJSON:        rulesCopy,
		searchJSON:       searchCopy,
		publicSearchJSON: publicSearchJSON,
		specIndex:        specIndex,
		sdk:              sdk,
		search:           search,
	}, nil
}
