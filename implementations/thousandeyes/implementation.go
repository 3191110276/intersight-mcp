package thousandeyes

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/thousandeyes/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "thousandeyes"
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
	return implementations.StandardGenerationConfig("thousandeyes", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "ThousandEyes",
		ServerName:      "thousandeyes-mcp",
		ConfigPrefix:    "THOUSANDEYES",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check THOUSANDEYES_BEARER_TOKEN or THOUSANDEYES_API_TOKEN.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "ThousandEyes Spec Search",
			SearchDescription: "Search the ThousandEyes discovery catalog for resources and operations.",
			QueryTitle:        "ThousandEyes Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "ThousandEyes Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the ThousandEyes API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "THOUSANDEYES_BEARER_TOKEN", Placeholder: "<THOUSANDEYES_BEARER_TOKEN>"},
				{EnvVarName: "THOUSANDEYES_API_TOKEN", Placeholder: "<THOUSANDEYES_API_TOKEN>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
