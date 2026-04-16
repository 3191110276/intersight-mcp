package catalystsdwan

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/catalyst-sdwan/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "catalyst-sdwan"
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
	return implementations.StandardGenerationConfig("catalyst-sdwan", implementations.StandardGenerationConfigOptions{})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Catalyst SD-WAN Manager",
		ServerName:      "catalyst-sdwan-mcp",
		ConfigPrefix:    "CATALYST_SDWAN",
		DefaultEndpoint: "",
		AuthErrorHint:   "Check CATALYST_SDWAN_ENDPOINT and the configured authentication variables.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Catalyst SD-WAN Spec Search",
			SearchDescription: "Search the Catalyst SD-WAN Manager discovery catalog for resources and operations.",
			QueryTitle:        "Catalyst SD-WAN Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Catalyst SD-WAN Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Catalyst SD-WAN Manager API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "CATALYST_SDWAN_PASSWORD", Placeholder: "<CATALYST_SDWAN_PASSWORD>"},
				{EnvVarName: "CATALYST_SDWAN_BEARER_TOKEN", Placeholder: "<CATALYST_SDWAN_BEARER_TOKEN>"},
				{EnvVarName: "CATALYST_SDWAN_SESSION_COOKIE", Placeholder: "<CATALYST_SDWAN_SESSION_COOKIE>"},
				{EnvVarName: "CATALYST_SDWAN_XSRF_TOKEN", Placeholder: "<CATALYST_SDWAN_XSRF_TOKEN>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
