package webex

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/webex/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "webex"
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
	return implementations.StandardGenerationConfig("webex", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Webex",
		ServerName:      "webex-mcp",
		ConfigPrefix:    "WEBEX",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check WEBEX_ACCESS_TOKEN or WEBEX_BEARER_TOKEN.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Webex Spec Search",
			SearchDescription: "Search the Webex discovery catalog for resources and operations.",
			QueryTitle:        "Webex Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Webex Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Webex API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "WEBEX_ACCESS_TOKEN", Placeholder: "<WEBEX_ACCESS_TOKEN>"},
				{EnvVarName: "WEBEX_BEARER_TOKEN", Placeholder: "<WEBEX_BEARER_TOKEN>"},
				{EnvVarName: "WEBEX_TOKEN", Placeholder: "<WEBEX_TOKEN>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
