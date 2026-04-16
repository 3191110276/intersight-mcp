package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mimaurer/intersight-mcp/internal/limits"
)

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
)

type RuntimeConfig struct {
	ReadOnly            bool
	LogLevel            LogLevel
	UnsafeLogFullCode   bool
	LegacyContentMirror bool
	Execution           limits.Execution
	SearchTimeout       time.Duration
	PerCallTimeout      time.Duration
	MaxCodeSize         int
	WASMMemory          uint64
}

func LoadRuntime(args []string, environ []string, envPrefix string) (RuntimeConfig, error) {
	cfg := RuntimeConfig{
		LogLevel: LogLevelInfo,
		Execution: limits.Execution{
			GlobalTimeout:  limits.DefaultGlobalTimeout,
			MaxAPICalls:    limits.DefaultMaxAPICalls,
			MaxOutputBytes: limits.DefaultMaxOutput,
			MaxConcurrent:  limits.DefaultMaxConcurrent,
		},
		SearchTimeout:  limits.SearchTimeout,
		PerCallTimeout: limits.PerCallTimeout,
		MaxCodeSize:    limits.MaxCodeSizeBytes,
		WASMMemory:     limits.WASMMemoryBytes,
	}

	env := ParseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var timeoutFlag string
	var maxAPICallsFlag int
	var maxOutputFlag string
	var maxConcurrentFlag int
	var logLevelFlag string
	var readOnlyFlag bool
	var unsafeLogFullCodeFlag bool
	var legacyContentMirrorFlag bool
	var searchTimeoutFlag string
	var perCallTimeoutFlag string
	var maxCodeSizeFlag string
	var wasmMemoryFlag string

	fs.StringVar(&timeoutFlag, "timeout", "", "global execution timeout")
	fs.IntVar(&maxAPICallsFlag, "max-api-calls", 0, "maximum API calls per execution")
	fs.StringVar(&maxOutputFlag, "max-output", "", "maximum serialized output size")
	fs.IntVar(&maxConcurrentFlag, "max-concurrent", 0, "maximum concurrent tool executions across search, query, and mutate")
	fs.StringVar(&logLevelFlag, "log-level", "", "log level: info or debug")
	fs.BoolVar(&readOnlyFlag, "read-only", false, "disable persistent write operations by omitting the mutate tool")
	fs.BoolVar(&unsafeLogFullCodeFlag, "unsafe-log-full-code", false, "include submitted tool code in debug logs with best-effort redaction; use only for short-lived incident debugging")
	fs.BoolVar(&legacyContentMirrorFlag, "legacy-content-mirror", false, "mirror full results into text content for legacy MCP clients")
	fs.StringVar(&searchTimeoutFlag, "search-timeout", "", "timeout for search executions")
	fs.StringVar(&perCallTimeoutFlag, "per-call-timeout", "", "timeout for individual HTTP and bootstrap calls")
	fs.StringVar(&maxCodeSizeFlag, "max-code-size", "", "maximum submitted code size")
	fs.StringVar(&wasmMemoryFlag, "wasm-memory", "", "QuickJS WebAssembly memory limit")

	if err := fs.Parse(FilterArgs(args, map[string]bool{
		"timeout":               true,
		"max-api-calls":         true,
		"max-output":            true,
		"max-concurrent":        true,
		"log-level":             true,
		"read-only":             false,
		"unsafe-log-full-code":  false,
		"legacy-content-mirror": false,
		"search-timeout":        true,
		"per-call-timeout":      true,
		"max-code-size":         true,
		"wasm-memory":           true,
	})); err != nil {
		return RuntimeConfig{}, err
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	timeoutRaw := env[envKey(envPrefix, "TIMEOUT")]
	if setFlags["timeout"] {
		timeoutRaw = timeoutFlag
	}
	if strings.TrimSpace(timeoutRaw) != "" {
		timeout, err := time.ParseDuration(strings.TrimSpace(timeoutRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid timeout %q: %w", timeoutRaw, err)
		}
		if timeout <= 0 {
			return RuntimeConfig{}, fmt.Errorf("invalid timeout %q: must be positive", timeoutRaw)
		}
		cfg.Execution.GlobalTimeout = timeout
	}

	searchTimeoutRaw := env[envKey(envPrefix, "SEARCH_TIMEOUT")]
	if setFlags["search-timeout"] {
		searchTimeoutRaw = searchTimeoutFlag
	}
	if strings.TrimSpace(searchTimeoutRaw) != "" {
		timeout, err := time.ParseDuration(strings.TrimSpace(searchTimeoutRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid search-timeout %q: %w", searchTimeoutRaw, err)
		}
		if timeout <= 0 {
			return RuntimeConfig{}, fmt.Errorf("invalid search-timeout %q: must be positive", searchTimeoutRaw)
		}
		cfg.SearchTimeout = timeout
	}

	perCallTimeoutRaw := env[envKey(envPrefix, "PER_CALL_TIMEOUT")]
	if setFlags["per-call-timeout"] {
		perCallTimeoutRaw = perCallTimeoutFlag
	}
	if strings.TrimSpace(perCallTimeoutRaw) != "" {
		timeout, err := time.ParseDuration(strings.TrimSpace(perCallTimeoutRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid per-call-timeout %q: %w", perCallTimeoutRaw, err)
		}
		if timeout <= 0 {
			return RuntimeConfig{}, fmt.Errorf("invalid per-call-timeout %q: must be positive", perCallTimeoutRaw)
		}
		cfg.PerCallTimeout = timeout
	}

	maxOutputRaw := env[envKey(envPrefix, "MAX_OUTPUT")]
	if setFlags["max-output"] {
		maxOutputRaw = maxOutputFlag
	}
	if strings.TrimSpace(maxOutputRaw) != "" {
		size, err := ParseByteSize(maxOutputRaw)
		if err != nil {
			return RuntimeConfig{}, err
		}
		cfg.Execution.MaxOutputBytes = size
	}

	maxAPICallsRaw := env[envKey(envPrefix, "MAX_API_CALLS")]
	if setFlags["max-api-calls"] {
		maxAPICallsRaw = strconv.Itoa(maxAPICallsFlag)
	}
	if strings.TrimSpace(maxAPICallsRaw) != "" {
		value, err := ParsePositiveInt("max-api-calls", maxAPICallsRaw)
		if err != nil {
			return RuntimeConfig{}, err
		}
		cfg.Execution.MaxAPICalls = value
	}

	maxConcurrentRaw := env[envKey(envPrefix, "MAX_CONCURRENT")]
	if setFlags["max-concurrent"] {
		maxConcurrentRaw = strconv.Itoa(maxConcurrentFlag)
	}
	if strings.TrimSpace(maxConcurrentRaw) != "" {
		value, err := ParsePositiveInt("max-concurrent", maxConcurrentRaw)
		if err != nil {
			return RuntimeConfig{}, err
		}
		cfg.Execution.MaxConcurrent = value
	}

	logLevelRaw := env[envKey(envPrefix, "LOG_LEVEL")]
	if setFlags["log-level"] {
		logLevelRaw = logLevelFlag
	}
	if strings.TrimSpace(logLevelRaw) != "" {
		switch LogLevel(strings.ToLower(strings.TrimSpace(logLevelRaw))) {
		case LogLevelInfo:
			cfg.LogLevel = LogLevelInfo
		case LogLevelDebug:
			cfg.LogLevel = LogLevelDebug
		default:
			return RuntimeConfig{}, fmt.Errorf("invalid log level %q: must be info or debug", logLevelRaw)
		}
	}

	readOnlyRaw := env[envKey(envPrefix, "READ_ONLY")]
	if setFlags["read-only"] {
		readOnlyRaw = strconv.FormatBool(readOnlyFlag)
	}
	if strings.TrimSpace(readOnlyRaw) != "" {
		value, err := strconv.ParseBool(strings.TrimSpace(readOnlyRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid read-only %q: must be true or false", readOnlyRaw)
		}
		cfg.ReadOnly = value
	}

	unsafeLogFullCodeRaw := env[envKey(envPrefix, "UNSAFE_LOG_FULL_CODE")]
	if setFlags["unsafe-log-full-code"] {
		unsafeLogFullCodeRaw = strconv.FormatBool(unsafeLogFullCodeFlag)
	}
	if strings.TrimSpace(unsafeLogFullCodeRaw) != "" {
		value, err := strconv.ParseBool(strings.TrimSpace(unsafeLogFullCodeRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid unsafe-log-full-code %q: must be true or false", unsafeLogFullCodeRaw)
		}
		cfg.UnsafeLogFullCode = value
	}

	legacyContentMirrorRaw := env[envKey(envPrefix, "LEGACY_CONTENT_MIRROR")]
	if setFlags["legacy-content-mirror"] {
		legacyContentMirrorRaw = strconv.FormatBool(legacyContentMirrorFlag)
	}
	if strings.TrimSpace(legacyContentMirrorRaw) != "" {
		value, err := strconv.ParseBool(strings.TrimSpace(legacyContentMirrorRaw))
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid legacy-content-mirror %q: must be true or false", legacyContentMirrorRaw)
		}
		cfg.LegacyContentMirror = value
	}

	maxCodeSizeRaw := env[envKey(envPrefix, "MAX_CODE_SIZE")]
	if setFlags["max-code-size"] {
		maxCodeSizeRaw = maxCodeSizeFlag
	}
	if strings.TrimSpace(maxCodeSizeRaw) != "" {
		size, err := ParsePositiveByteSizeInt("max-code-size", maxCodeSizeRaw)
		if err != nil {
			return RuntimeConfig{}, err
		}
		cfg.MaxCodeSize = size
	}

	wasmMemoryRaw := env[envKey(envPrefix, "WASM_MEMORY")]
	if setFlags["wasm-memory"] {
		wasmMemoryRaw = wasmMemoryFlag
	}
	if strings.TrimSpace(wasmMemoryRaw) != "" {
		size, err := ParsePositiveByteSizeUint64("wasm-memory", wasmMemoryRaw)
		if err != nil {
			return RuntimeConfig{}, err
		}
		cfg.WASMMemory = size
	}

	return cfg, nil
}

func (c RuntimeConfig) DebugLoggingEnabled() bool {
	return c.LogLevel == LogLevelDebug
}

func ParseEnv(environ []string) map[string]string {
	parsed := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		parsed[key] = value
	}
	return parsed
}

func ParsePositiveInt(name, raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid %s %q: must be a positive integer", name, raw)
	}
	return value, nil
}

func ParseByteSize(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("invalid max-output \"\": value is required")
	}

	type suffix struct {
		label string
		scale int64
	}

	suffixes := []suffix{
		{label: "KB", scale: 1024},
		{label: "MB", scale: 1024 * 1024},
		{label: "GB", scale: 1024 * 1024 * 1024},
	}
	upper := strings.ToUpper(raw)
	for _, suffix := range suffixes {
		if strings.HasSuffix(upper, suffix.label) {
			base := strings.TrimSpace(upper[:len(upper)-len(suffix.label)])
			value, err := strconv.ParseInt(base, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid max-output %q: %w", raw, err)
			}
			if value <= 0 {
				return 0, fmt.Errorf("invalid max-output %q: must be positive", raw)
			}
			return value * suffix.scale, nil
		}
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid max-output %q: expected bytes or a binary suffix such as 512KB", raw)
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid max-output %q: must be positive", raw)
	}
	return value, nil
}

func ParsePositiveByteSizeInt(name, raw string) (int, error) {
	size, err := ParseByteSize(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	const maxInt = int(^uint(0) >> 1)
	if size > int64(maxInt) {
		return 0, fmt.Errorf("invalid %s %q: exceeds platform integer size", name, raw)
	}
	return int(size), nil
}

func ParsePositiveByteSizeUint64(name, raw string) (uint64, error) {
	size, err := ParseByteSize(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return uint64(size), nil
}

func envKey(prefix, suffix string) string {
	prefix = strings.TrimSpace(prefix)
	suffix = strings.TrimSpace(suffix)
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "_" + suffix
	}
}

func FilterArgs(args []string, known map[string]bool) []string {
	if len(args) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if key, _, ok := strings.Cut(name, "="); ok {
			name = key
		}
		expectsValue, ok := known[name]
		if !ok {
			continue
		}
		filtered = append(filtered, args[i])
		if expectsValue && !strings.Contains(args[i], "=") && i+1 < len(args) {
			filtered = append(filtered, args[i+1])
			i++
		}
	}
	return filtered
}
