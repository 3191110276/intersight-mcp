package implementations

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

// Artifacts holds the generated startup artifacts for a specific implementation
// target. The byte slices returned by a target package should be treated as
// immutable snapshots.
type Artifacts struct {
	ResolvedSpec  []byte
	SDKCatalog    []byte
	Rules         []byte
	SearchCatalog []byte
}

type ToolDescriptions struct {
	SearchTitle       string
	SearchDescription string
	QueryTitle        string
	QueryDescription  string
	MutateTitle       string
	MutateDescription string
}

type LogRedaction struct {
	EnvVarName  string
	Placeholder string
}

type LoggingMetadata struct {
	Redactions []LogRedaction
}

type RuntimeMetadata struct {
	ProviderName     string
	ServerName       string
	ConfigPrefix     string
	DefaultEndpoint  string
	AuthErrorHint    string
	ToolDescriptions ToolDescriptions
	Logging          LoggingMetadata
}

type GenerationConfig struct {
	RawSpecPath             string
	ManifestPath            string
	FilterPath              string
	MetricsPath             string
	OutputPath              string
	FallbackPathPrefixes    []string
	RuleTemplates           []contracts.RuleTemplate
	SchemaNormalizationHook SchemaNormalizationHook
}

type SchemaNormalizationHook func(proxy *highbase.SchemaProxy, schema *highbase.Schema, out *contracts.NormalizedSchema)

type StandardGenerationConfigOptions struct {
	IncludeFilter           bool
	IncludeMetrics          bool
	FallbackPathPrefixes    []string
	RuleTemplates           []contracts.RuleTemplate
	SchemaNormalizationHook SchemaNormalizationHook
}

func StandardGenerationConfig(provider string, opts StandardGenerationConfigOptions) GenerationConfig {
	base := fmt.Sprintf("third_party/%s", provider)
	cfg := GenerationConfig{
		RawSpecPath:             fmt.Sprintf("%s/openapi/raw/openapi.json", base),
		ManifestPath:            fmt.Sprintf("%s/openapi/manifest.json", base),
		OutputPath:              fmt.Sprintf("implementations/%s/generated/spec_resolved.json", provider),
		FallbackPathPrefixes:    append([]string(nil), opts.FallbackPathPrefixes...),
		RuleTemplates:           append([]contracts.RuleTemplate(nil), opts.RuleTemplates...),
		SchemaNormalizationHook: opts.SchemaNormalizationHook,
	}
	if opts.IncludeFilter {
		cfg.FilterPath = fmt.Sprintf("implementations/%s/filter.yaml", provider)
	}
	if opts.IncludeMetrics {
		cfg.MetricsPath = fmt.Sprintf("%s/metrics/search_metrics.json", base)
	}
	return cfg
}

type APICaller interface {
	Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error)
}

type ConnectionConfig interface {
	ProxyURL() string
	NewAPICaller(ctx context.Context, timeout time.Duration, httpClient *http.Client) APICaller
}

type Target interface {
	Name() string
	RuntimeMetadata() RuntimeMetadata
	Artifacts() Artifacts
	GenerationConfig() GenerationConfig
	SandboxExtensions() providerext.Extensions
	LoadConnectionConfig(args []string, environ []string) (ConnectionConfig, error)
}
