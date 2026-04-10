package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

var validOperationMethods = map[string]struct{}{
	"get":     {},
	"post":    {},
	"put":     {},
	"patch":   {},
	"delete":  {},
	"head":    {},
	"options": {},
	"trace":   {},
}

func ValidateEmbeddedSpec(data []byte) error {
	var spec contracts.NormalizedSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("parse embedded spec: %w", err)
	}
	if spec.Metadata.PublishedVersion == "" || spec.Metadata.SHA256 == "" {
		return errors.New("embedded spec validation failed: source metadata is required")
	}
	if len(spec.Paths) == 0 {
		return errors.New("embedded spec validation failed: paths must be non-empty")
	}
	if len(spec.Schemas) == 0 {
		return errors.New("embedded spec validation failed: schemas must be present")
	}

	operationCount := 0
	hasAPIV1Path := false
	for path, methods := range spec.Paths {
		if strings.HasPrefix(path, "/api/v1/") {
			hasAPIV1Path = true
		}
		for method := range methods {
			if _, ok := validOperationMethods[strings.ToLower(method)]; ok {
				operationCount++
			}
		}
	}
	if operationCount == 0 {
		return errors.New("embedded spec validation failed: at least one valid operation is required")
	}
	if !hasAPIV1Path {
		return errors.New("embedded spec validation failed: expected at least one /api/v1/ path")
	}
	if _, ok := spec.Schemas["compute.RackUnit"]; !ok {
		return errors.New("embedded spec validation failed: compute.RackUnit schema is required")
	}
	return nil
}

func ValidateEmbeddedArtifacts(specData, catalogData, rulesData, searchCatalogData []byte) error {
	if err := ValidateEmbeddedSpec(specData); err != nil {
		return err
	}

	var spec contracts.NormalizedSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return fmt.Errorf("parse embedded spec: %w", err)
	}
	var catalog contracts.SDKCatalog
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		return fmt.Errorf("parse embedded sdk catalog: %w", err)
	}
	if len(catalog.Methods) == 0 {
		return errors.New("embedded artifact validation failed: sdk catalog methods must be non-empty")
	}
	if catalog.Metadata.PublishedVersion == "" || catalog.Metadata.SHA256 == "" {
		return errors.New("embedded artifact validation failed: sdk catalog source metadata is required")
	}
	if err := contracts.ValidateSDKCatalogAgainstSpec(spec, catalog); err != nil {
		return err
	}

	var rules contracts.RuleCatalog
	if err := json.Unmarshal(rulesData, &rules); err != nil {
		return fmt.Errorf("parse embedded rules: %w", err)
	}
	if rules.Metadata.PublishedVersion == "" || rules.Metadata.SHA256 == "" {
		return errors.New("embedded artifact validation failed: rule metadata source metadata is required")
	}
	if err := contracts.ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules); err != nil {
		return err
	}

	var searchCatalog contracts.SearchCatalog
	if err := json.Unmarshal(searchCatalogData, &searchCatalog); err != nil {
		return fmt.Errorf("parse embedded search catalog: %w", err)
	}
	if searchCatalog.Metadata.PublishedVersion == "" || searchCatalog.Metadata.SHA256 == "" {
		return errors.New("embedded artifact validation failed: search catalog source metadata is required")
	}
	if err := contracts.ValidateSearchCatalogAgainstArtifacts(spec, catalog, rules, searchCatalog); err != nil {
		return err
	}
	return nil
}
