package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
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

type Config struct {
	Endpoint       string
	Origin         string
	OAuthTokenURL  string
	APIBaseURL     string
	ClientID       string
	ClientSecret   string
	LogLevel       LogLevel
	Execution      limits.Execution
	SearchTimeout  time.Duration
	PerCallTimeout time.Duration
	MaxCodeSize    int
	WASMMemory     uint64
}

func Load(args []string, environ []string) (Config, error) {
	cfg := Config{
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

	env := parseEnv(environ)

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var endpointFlag string
	var timeoutFlag string
	var maxAPICallsFlag int
	var maxOutputFlag string
	var maxConcurrentFlag int
	var logLevelFlag string

	fs.StringVar(&endpointFlag, "endpoint", "", "base Intersight endpoint origin")
	fs.StringVar(&timeoutFlag, "timeout", "", "global execution timeout")
	fs.IntVar(&maxAPICallsFlag, "max-api-calls", 0, "maximum API calls per execution")
	fs.StringVar(&maxOutputFlag, "max-output", "", "maximum serialized output size")
	fs.IntVar(&maxConcurrentFlag, "max-concurrent", 0, "maximum concurrent query/mutate executions")
	fs.StringVar(&logLevelFlag, "log-level", "", "log level: info or debug")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	cfg.ClientID = strings.TrimSpace(env["INTERSIGHT_CLIENT_ID"])
	cfg.ClientSecret = strings.TrimSpace(env["INTERSIGHT_CLIENT_SECRET"])
	if cfg.ClientID == "" {
		return Config{}, errors.New("missing required INTERSIGHT_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		return Config{}, errors.New("missing required INTERSIGHT_CLIENT_SECRET")
	}

	endpointRaw := limits.DefaultEndpoint
	if value := strings.TrimSpace(env["INTERSIGHT_ENDPOINT"]); value != "" {
		endpointRaw = value
	}
	if setFlags["endpoint"] {
		endpointRaw = strings.TrimSpace(endpointFlag)
	}

	parsedEndpoint, err := validateEndpoint(endpointRaw)
	if err != nil {
		return Config{}, err
	}
	cfg.Endpoint = parsedEndpoint.String()
	cfg.Origin = parsedEndpoint.Scheme + "://" + parsedEndpoint.Host
	cfg.OAuthTokenURL = cfg.Origin + "/iam/token"
	cfg.APIBaseURL = cfg.Origin + "/api/v1"

	timeoutRaw := env["INTERSIGHT_TIMEOUT"]
	if setFlags["timeout"] {
		timeoutRaw = timeoutFlag
	}
	if strings.TrimSpace(timeoutRaw) != "" {
		timeout, err := time.ParseDuration(strings.TrimSpace(timeoutRaw))
		if err != nil {
			return Config{}, fmt.Errorf("invalid timeout %q: %w", timeoutRaw, err)
		}
		if timeout <= 0 {
			return Config{}, fmt.Errorf("invalid timeout %q: must be positive", timeoutRaw)
		}
		cfg.Execution.GlobalTimeout = timeout
	}

	maxOutputRaw := env["INTERSIGHT_MAX_OUTPUT"]
	if setFlags["max-output"] {
		maxOutputRaw = maxOutputFlag
	}
	if strings.TrimSpace(maxOutputRaw) != "" {
		size, err := parseByteSize(maxOutputRaw)
		if err != nil {
			return Config{}, err
		}
		cfg.Execution.MaxOutputBytes = size
	}

	maxAPICallsRaw := env["INTERSIGHT_MAX_API_CALLS"]
	if setFlags["max-api-calls"] {
		maxAPICallsRaw = strconv.Itoa(maxAPICallsFlag)
	}
	if strings.TrimSpace(maxAPICallsRaw) != "" {
		value, err := parsePositiveInt("max-api-calls", maxAPICallsRaw)
		if err != nil {
			return Config{}, err
		}
		cfg.Execution.MaxAPICalls = value
	}

	maxConcurrentRaw := env["INTERSIGHT_MAX_CONCURRENT"]
	if setFlags["max-concurrent"] {
		maxConcurrentRaw = strconv.Itoa(maxConcurrentFlag)
	}
	if strings.TrimSpace(maxConcurrentRaw) != "" {
		value, err := parsePositiveInt("max-concurrent", maxConcurrentRaw)
		if err != nil {
			return Config{}, err
		}
		cfg.Execution.MaxConcurrent = value
	}

	logLevelRaw := env["INTERSIGHT_LOG_LEVEL"]
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
			return Config{}, fmt.Errorf("invalid log level %q: must be info or debug", logLevelRaw)
		}
	}

	return cfg, nil
}

func (c Config) DebugLoggingEnabled() bool {
	return c.LogLevel == LogLevelDebug
}

func validateEndpoint(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("invalid endpoint %q: must be an absolute URL", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid endpoint %q: scheme must be http or https", raw)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid endpoint %q: host is required", raw)
	}
	if parsed.RawQuery != "" {
		return nil, fmt.Errorf("invalid endpoint %q: query is not allowed", raw)
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid endpoint %q: fragment is not allowed", raw)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, fmt.Errorf("invalid endpoint %q: path is not allowed; use the origin only", raw)
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed, nil
}

func parsePositiveInt(name, raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid %s %q: must be a positive integer", name, raw)
	}
	return value, nil
}

func parseByteSize(raw string) (int64, error) {
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

func parseEnv(environ []string) map[string]string {
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
