package intersight

import (
	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/implementations/intersight/generated"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type target struct{}

func init() {
	implementations.RegisterTarget(target{})
}

func Target() implementations.Target {
	return target{}
}

func Artifacts() implementations.Artifacts {
	return Target().Artifacts()
}

func (target) Name() string {
	return "intersight"
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
	return implementations.StandardGenerationConfig("intersight", implementations.StandardGenerationConfigOptions{
		IncludeFilter:           true,
		IncludeMetrics:          true,
		FallbackPathPrefixes:    []string{"/api/v1/"},
		RuleTemplates:           RuleTemplates(),
		SchemaNormalizationHook: SchemaNormalizationHook(),
	})
}

func (target) SandboxExtensions() providerext.Extensions {
	return SandboxExtensions()
}

func (target) RuntimeMetadata() implementations.RuntimeMetadata {
	return implementations.RuntimeMetadata{
		ProviderName:    "Intersight",
		ServerName:      "intersight-mcp",
		ConfigPrefix:    "INTERSIGHT",
		DefaultEndpoint: defaultEndpoint,
		AuthErrorHint:   "Check INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET.",
		ToolDescriptions: implementations.ToolDescriptions{
			SearchTitle:       "Intersight Spec Search",
			SearchDescription: searchDescription,
			QueryTitle:        "Intersight Query",
			QueryDescription:  queryDescription,
			MutateTitle:       "Intersight Mutate",
			MutateDescription: mutateDescription,
		},
		Logging: implementations.LoggingMetadata{
			Redactions: []implementations.LogRedaction{
				{EnvVarName: "INTERSIGHT_CLIENT_SECRET", Placeholder: "<CLIENT_SECRET>"},
				{EnvVarName: "INTERSIGHT_CLIENT_ID", Placeholder: "<CLIENT_ID>"},
			},
		},
	}
}

func (target) LoadConnectionConfig(args []string, environ []string) (implementations.ConnectionConfig, error) {
	return LoadConnectionConfig(args, environ)
}
