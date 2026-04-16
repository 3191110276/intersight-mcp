package catalystcenter

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/catalyst-center/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "catalyst-center"
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
	return implementations.StandardGenerationConfig("catalyst-center", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Catalyst Center",
		ServerName:      "catalyst-center-mcp",
		ConfigPrefix:    "CATALYST_CENTER",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check CATALYST_CENTER_ENDPOINT plus either CATALYST_CENTER_X_AUTH_TOKEN or CATALYST_CENTER_USERNAME and CATALYST_CENTER_PASSWORD.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Catalyst Center Spec Search",
			SearchDescription: "Search the Catalyst Center discovery catalog for resources and operations.",
			QueryTitle:        "Catalyst Center Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Catalyst Center Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Catalyst Center API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "CATALYST_CENTER_PASSWORD", Placeholder: "[CATALYST_CENTER_PASSWORD]"},
				{EnvVarName: "CATALYST_CENTER_X_AUTH_TOKEN", Placeholder: "[CATALYST_CENTER_X_AUTH_TOKEN]"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
