package limits

import "time"

const (
	DefaultEndpoint      = "https://intersight.com"
	DefaultGlobalTimeout = 40 * time.Second
	DefaultMaxAPICalls   = 250
	DefaultMaxOutput     = 512 * 1024
	DefaultMaxConcurrent = 50

	SearchTimeout    = 15 * time.Second
	PerCallTimeout   = 15 * time.Second
	MaxCodeSizeBytes = 100 * 1024
	WASMMemoryBytes  = 64 * 1024 * 1024
)

type Execution struct {
	GlobalTimeout  time.Duration
	MaxAPICalls    int
	MaxOutputBytes int64
	MaxConcurrent  int
}

func DefaultExecution() Execution {
	return Execution{
		GlobalTimeout:  DefaultGlobalTimeout,
		MaxAPICalls:    DefaultMaxAPICalls,
		MaxOutputBytes: DefaultMaxOutput,
		MaxConcurrent:  DefaultMaxConcurrent,
	}
}
