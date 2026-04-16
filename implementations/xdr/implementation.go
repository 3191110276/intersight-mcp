package xdr

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/xdr/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "xdr"
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
	return implementations.StandardGenerationConfig("xdr", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Cisco XDR",
		ServerName:      "xdr-mcp",
		ConfigPrefix:    "XDR",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check XDR_ACCESS_TOKEN or XDR_CLIENT_ID and XDR_CLIENT_SECRET.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Cisco XDR Spec Search",
			SearchDescription: "Search the Cisco XDR discovery catalog for resources and operations.",
			QueryTitle:        "Cisco XDR Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Cisco XDR Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Cisco XDR API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "XDR_ACCESS_TOKEN", Placeholder: "<XDR_ACCESS_TOKEN>"},
				{EnvVarName: "XDR_API_TOKEN", Placeholder: "<XDR_API_TOKEN>"},
				{EnvVarName: "XDR_CLIENT_ID", Placeholder: "<XDR_CLIENT_ID>"},
				{EnvVarName: "XDR_CLIENT_SECRET", Placeholder: "<XDR_CLIENT_SECRET>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
