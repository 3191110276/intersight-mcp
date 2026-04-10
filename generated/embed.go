package generated

import _ "embed"

//go:generate sh -c "mkdir -p ../.cache/go-build ../.tmp && GOCACHE=$(cd .. && pwd)/.cache/go-build GOTMPDIR=$(cd .. && pwd)/.tmp go -C .. run ./cmd/generate --in third_party/intersight/openapi/raw/openapi.json --filter spec/filter.yaml --metrics third_party/intersight/metrics/search_metrics.json --out generated/spec_resolved.json"

//go:embed spec_resolved.json
var specResolvedJSON []byte

//go:embed sdk_catalog.json
var sdkCatalogJSON []byte

//go:embed rules.json
var rulesJSON []byte

//go:embed search_catalog.json
var searchCatalogJSON []byte

func ResolvedSpecBytes() []byte {
	return append([]byte(nil), specResolvedJSON...)
}

func SDKCatalogBytes() []byte {
	return append([]byte(nil), sdkCatalogJSON...)
}

func RulesBytes() []byte {
	return append([]byte(nil), rulesJSON...)
}

func SearchCatalogBytes() []byte {
	return append([]byte(nil), searchCatalogJSON...)
}
