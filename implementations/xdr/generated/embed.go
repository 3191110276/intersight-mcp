package generated

import _ "embed"

//go:generate sh -c "mkdir -p ../../../.cache/go-build ../../../.tmp && GOCACHE=$(cd ../../.. && pwd)/.cache/go-build GOTMPDIR=$(cd ../../.. && pwd)/.tmp go -C ../../.. run ./cmd/generate --provider xdr"

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
