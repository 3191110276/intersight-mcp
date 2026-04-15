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
	operationCount := 0
	for _, methods := range spec.Paths {
		for method := range methods {
			if _, ok := validOperationMethods[strings.ToLower(method)]; ok {
				operationCount++
			}
		}
	}
	if operationCount == 0 {
		return errors.New("embedded spec validation failed: at least one valid operation is required")
	}
	return nil
}

func ValidateEmbeddedArtifacts(specData, catalogData, rulesData, searchCatalogData []byte) error {
	return ValidateEmbeddedArtifactsWithRuleTemplates(specData, catalogData, rulesData, searchCatalogData, nil)
}

func ValidateEmbeddedArtifactsWithRuleTemplates(specData, catalogData, rulesData, searchCatalogData []byte, ruleTemplates []contracts.RuleTemplate) error {
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
	if err := contracts.ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules, ruleTemplates); err != nil {
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
