package nexusdashboard

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/nexus-dashboard/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func (target) Name() string {
	return "nexus-dashboard"
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
	return implementations.StandardGenerationConfig("nexus-dashboard", implementations.StandardGenerationConfigOptions{
		IncludeFilter:  false,
		IncludeMetrics: false,
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{}
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Nexus Dashboard",
		ServerName:      "nexus-dashboard-mcp",
		ConfigPrefix:    "NEXUS_DASHBOARD",
		DefaultEndpoint: "https://<cluster>",
		AuthErrorHint:   "Check NEXUS_DASHBOARD endpoint and credentials. Supported auth modes are bearer token, username plus API key, or username plus password.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Nexus Dashboard Spec Search",
			SearchDescription: "Search the Nexus Dashboard discovery catalog for resources and operations.",
			QueryTitle:        "Nexus Dashboard Query",
			QueryDescription:  "Run read-shaped SDK methods or offline validation for write-shaped methods.",
			MutateTitle:       "Nexus Dashboard Mutate",
			MutateDescription: "Run persistent write-shaped SDK methods against the Nexus Dashboard API.",
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "NEXUS_DASHBOARD_PASSWORD", Placeholder: "<NEXUS_DASHBOARD_PASSWORD>"},
				{EnvVarName: "NEXUS_DASHBOARD_API_KEY", Placeholder: "<NEXUS_DASHBOARD_API_KEY>"},
				{EnvVarName: "NEXUS_DASHBOARD_BEARER_TOKEN", Placeholder: "<NEXUS_DASHBOARD_BEARER_TOKEN>"},
				{EnvVarName: "NEXUS_DASHBOARD_TOKEN", Placeholder: "<NEXUS_DASHBOARD_TOKEN>"},
				{EnvVarName: "NEXUS_DASHBOARD_AUTH_COOKIE", Placeholder: "<NEXUS_DASHBOARD_AUTH_COOKIE>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
