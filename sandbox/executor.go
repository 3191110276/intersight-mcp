package sandbox

import (
	"context"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/limits"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

type Mode string

const (
	ModeSearch   Mode = "search"
	ModeQuery    Mode = "query"
	ModeValidate Mode = "validate"
	ModeMutate   Mode = "mutate"
)

type Result struct {
	Value        any
	Logs         []string
	APICallCount int
	Presentation *PresentationHint
}

type PresentationHint = providerext.PresentationHint

const (
	PresentationKindMetricsImage = "metrics-image"
	PresentationKindMetricsApp   = "metrics-app"
)

type Executor interface {
	Execute(ctx context.Context, code string, mode Mode) (Result, error)
	Close() error
}

type APICaller interface {
	Do(ctx context.Context, operation contracts.OperationDescriptor) (any, error)
}

type APIRequestOptions struct {
	Query       map[string]string
	Headers     map[string]string
	Body        any
	DryRun      bool
	EndpointURL string
}

type Config struct {
	SearchTimeout     time.Duration
	GlobalTimeout     time.Duration
	PerCallTimeout    time.Duration
	MaxCodeSize       int
	MaxAPICalls       int
	MaxOutputBytes    int64
	WASMMemoryBytes   int
	EnableMetricsApps bool
}

func DefaultConfig() Config {
	return Config{
		SearchTimeout:   limits.SearchTimeout,
		GlobalTimeout:   limits.DefaultGlobalTimeout,
		PerCallTimeout:  limits.PerCallTimeout,
		MaxCodeSize:     limits.MaxCodeSizeBytes,
		MaxAPICalls:     limits.DefaultMaxAPICalls,
		MaxOutputBytes:  limits.DefaultMaxOutput,
		WASMMemoryBytes: int(limits.WASMMemoryBytes),
	}
}
