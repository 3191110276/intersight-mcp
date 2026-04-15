package meraki

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/meraki/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "meraki"
}

func (target) Artifacts() implementations.Artifacts {
	return implementations.Artifacts{
		ResolvedSpec:  generated.ResolvedSpecBytes(),
		SDKCatalog:    generated.SDKCatalogBytes(),
		Rules:         generated.RulesBytes(),
		SearchCatalog: generated.SearchCatalogBytes(),
	}
}

func (target) GenerationConfig() implementations.GenerationConfig {
	return implementations.StandardGenerationConfig("meraki", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return SandboxExtensions()
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Meraki",
		ServerName:      "meraki-mcp",
		ConfigPrefix:    "MERAKI",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check MERAKI_API_KEY or MERAKI_DASHBOARD_API_KEY.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Meraki Spec Search",
			SearchDescription: "Search the Meraki discovery catalog for resources and operations.",
			QueryTitle:        "Meraki Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Meraki Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Meraki API.",
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
