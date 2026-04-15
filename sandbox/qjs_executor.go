package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type qjsExecutor struct {
	cfg     Config
	client  APICaller
	spec    *dryRunSpecIndex
	sdk     *sdkRuntime
	initErr error
}

func NewQJSExecutor(cfg Config, client APICaller) Executor {
	return &qjsExecutor{
		cfg:     normalizeConfig(cfg),
		client:  client,
		initErr: errors.New("embedded sandbox artifacts are not configured; use NewQJSExecutorFromBundle or NewQJSExecutorWithArtifacts"),
	}
}

func NewQJSExecutorWithSpec(cfg Config, client APICaller, specJSON []byte) (Executor, error) {
	return NewQJSExecutorWithSpecAndExtensions(cfg, client, specJSON, Extensions{})
}

func NewQJSExecutorWithSpecAndExtensions(cfg Config, client APICaller, specJSON []byte, ext Extensions) (Executor, error) {
	spec, err := loadDryRunSpecIndex(specJSON, ext)
	if err != nil {
		return nil, err
	}
	return &qjsExecutor{
		cfg:    normalizeConfig(cfg),
		client: client,
		spec:   spec,
	}, nil
}

func NewQJSExecutorWithArtifacts(cfg Config, client APICaller, specJSON, catalogJSON, rulesJSON []byte) (Executor, error) {
	return NewQJSExecutorWithArtifactsAndExtensions(cfg, client, specJSON, catalogJSON, rulesJSON, Extensions{})
}

func NewQJSExecutorWithArtifactsAndExtensions(cfg Config, client APICaller, specJSON, catalogJSON, rulesJSON []byte, ext Extensions) (Executor, error) {
	spec, err := loadDryRunSpecIndex(specJSON, ext)
	if err != nil {
		return nil, err
	}
	sdk, err := loadSDKRuntime(specJSON, catalogJSON, rulesJSON, ext)
	if err != nil {
		return nil, err
	}
	return &qjsExecutor{
		cfg:    normalizeConfig(cfg),
		client: client,
		spec:   spec,
		sdk:    sdk,
	}, nil
}

func NewQJSExecutorFromBundle(cfg Config, client APICaller, bundle *ArtifactBundle) (Executor, error) {
	if bundle == nil {
		return nil, contracts.ValidationError{Message: "artifact bundle is required"}
	}
	return &qjsExecutor{
		cfg:    normalizeConfig(cfg),
		client: client,
		spec:   bundle.specIndex,
		sdk:    bundle.sdk,
	}, nil
}

func (e *qjsExecutor) Execute(ctx context.Context, code string, mode Mode) (Result, error) {
	if e.initErr != nil {
		return Result{}, contracts.InternalError{Message: "initialize embedded sandbox artifacts", Err: e.initErr}
	}
	if mode != ModeSearch && e.sdk == nil {
		return Result{}, contracts.ValidationError{Message: "sdk runtime is not configured for query or mutate execution"}
	}

	timeout := e.cfg.GlobalTimeout
	if mode == ModeSearch {
		timeout = e.cfg.SearchTimeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logs := newLogBuffer(e.cfg.MaxOutputBytes)
	rt, err := qjs.New(qjs.Option{
		Context:            execCtx,
		CloseOnContextDone: true,
		MemoryLimit:        e.cfg.WASMMemoryBytes,
		Stdout:             logs,
		Stderr:             logs,
	})
	if err != nil {
		return Result{}, contracts.InternalError{Message: "create QuickJS runtime", Err: err}
	}
	defer func() {
		defer func() {
			_ = recover()
		}()
		rt.Close()
	}()

	bridge := &apiBridge{
		client:            e.client,
		mode:              mode,
		perCallTimeout:    e.cfg.PerCallTimeout,
		maxAPICalls:       e.cfg.MaxAPICalls,
		enableMetricsApps: e.cfg.EnableMetricsApps,
		spec:              e.spec,
		sdk:               e.sdk,
	}
	if mode != ModeSearch {
		if err := e.sdk.install(rt.Context(), execCtx, bridge); err != nil {
			return Result{}, err
		}
	}

	result, err := executeWithRuntime(execCtx, rt, code, mode, e.cfg.MaxCodeSize, e.cfg.MaxOutputBytes)
	if err != nil {
		if len(result.Logs) == 0 {
			result.Logs = logs.Lines()
		}
		result.APICallCount = bridge.APICallCount()
		result.Presentation = bridge.presentation
		return result, err
	}
	if len(result.Logs) == 0 {
		result.Logs = logs.Lines()
	}
	result.APICallCount = bridge.APICallCount()
	result.Presentation = bridge.presentation
	return result, nil
}

func (e *qjsExecutor) Close() error {
	return nil
}

func normalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.SearchTimeout <= 0 {
		cfg.SearchTimeout = def.SearchTimeout
	}
	if cfg.GlobalTimeout <= 0 {
		cfg.GlobalTimeout = def.GlobalTimeout
	}
	if cfg.PerCallTimeout <= 0 {
		cfg.PerCallTimeout = def.PerCallTimeout
	}
	if cfg.MaxCodeSize <= 0 {
		cfg.MaxCodeSize = def.MaxCodeSize
	}
	if cfg.MaxAPICalls <= 0 {
		cfg.MaxAPICalls = def.MaxAPICalls
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = def.MaxOutputBytes
	}
	if cfg.WASMMemoryBytes <= 0 {
		cfg.WASMMemoryBytes = def.WASMMemoryBytes
	}
	return cfg
}

func executeWithRuntime(
	execCtx context.Context,
	rt *qjs.Runtime,
	code string,
	mode Mode,
	maxCodeSize int,
	maxOutputBytes int64,
) (result Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = normalizePanic(execCtx, recovered)
		}
	}()

	if len(code) > maxCodeSize {
		return Result{}, contracts.ValidationError{
			Message: fmt.Sprintf("submitted code exceeds the %d byte limit", maxCodeSize),
		}
	}

	script := wrappedUserScript(code)
	val, err := rt.Context().Eval("user_code.js", qjs.Code(script))
	if err != nil {
		return Result{}, normalizeJSError(execCtx, err)
	}
	defer val.Free()

	resolved := val
	if val.IsPromise() {
		resolved, err = val.Await()
		if err != nil {
			return Result{}, normalizeJSError(execCtx, err)
		}
		defer resolved.Free()
	}

	raw, err := qjs.ToGoValue[map[string]any](resolved)
	if err != nil {
		return Result{}, contracts.InternalError{Message: "decode sandbox result", Err: err}
	}

	logs := anyToStrings(raw["logs"])
	if errPayload, ok := raw["__executor_error__"]; ok && errPayload != nil {
		return Result{Logs: logs}, normalizeThrown(execCtx, errPayload)
	}

	value := raw["value"]
	size, err := serializedSize(map[string]any{
		"value": value,
		"logs":  logs,
	})
	if err != nil {
		return Result{}, contracts.InternalError{Message: "serialize sandbox result", Err: err}
	}
	if size > maxOutputBytes {
		return Result{}, contracts.OutputTooLarge{
			Message: fmt.Sprintf("Result serialized to %s, which exceeds the %s limit. Reduce the result set with $select, $top, or $filter.", humanBytes(size), humanBytes(maxOutputBytes)),
			Details: map[string]any{
				"bytes": size,
				"limit": maxOutputBytes,
				"mode":  string(mode),
			},
		}
	}

	return Result{Value: value, Logs: logs}, nil
}

func wrappedUserScript(code string) string {
	return `(async () => {
  const __logs = [];
  const __console = {
    log(...args) {
      __logs.push(args.map(arg => {
        if (typeof arg === 'string') {
          return arg;
        }
        try {
          return JSON.stringify(arg);
        } catch (_) {
          return String(arg);
        }
      }).join(' '));
    }
  };

  function __copyOwnProperties(value) {
    if (!value || typeof value !== 'object') {
      return value;
    }
    if (Array.isArray(value)) {
      return value.map(__copyOwnProperties);
    }
    const out = {};
    for (const key of Object.keys(value)) {
      try {
        out[key] = __copyOwnProperties(value[key]);
      } catch (_) {
        out[key] = '[unserializable]';
      }
    }
    return out;
  }

  try {
    return { logs: __logs, value: await (async (console) => {
` + code + `
    })(__console) };
  } catch (err) {
    const payload = { message: String(err) };
    if (err && typeof err === 'object') {
      payload.name = err.name || '';
      payload.error = __copyOwnProperties(err);
      if (typeof err.message === 'string') {
        payload.message = err.message;
      }
      if (typeof err.stack === 'string') {
        payload.stack = err.stack;
      }
    }
    return { logs: __logs, __executor_error__: payload };
  }
})()`
}

func normalizeThrown(execCtx context.Context, payload any) error {
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return contracts.TimeoutError{Message: "execution timed out"}
	}
	if errors.Is(execCtx.Err(), context.Canceled) {
		return contracts.InternalError{Message: "execution canceled", Err: execCtx.Err()}
	}

	raw, ok := payload.(map[string]any)
	if !ok {
		return contracts.ValidationError{Message: fmt.Sprintf("JavaScript execution failed: %v", payload)}
	}

	name, _ := raw["name"].(string)
	message, _ := raw["message"].(string)
	if message == "" {
		message = "JavaScript execution failed"
	}

	if errorPayload, ok := raw["error"].(map[string]any); ok && len(errorPayload) > 0 {
		if kind, _ := errorPayload["kind"].(string); kind != "" {
			switch kind {
			case "auth":
				return contracts.AuthError{Message: stringOr(errorPayload["message"], message)}
			case "http":
				status := numberToInt(errorPayload["status"])
				return contracts.HTTPError{
					Status:  status,
					Body:    errorPayload["body"],
					Message: fmt.Sprintf("API returned HTTP %d", status),
				}
			case "network":
				return contracts.NetworkError{Message: stringOr(errorPayload["message"], message)}
			case "timeout":
				return contracts.TimeoutError{Message: stringOr(errorPayload["message"], message)}
			case "limit":
				return contracts.LimitError{Message: stringOr(errorPayload["message"], message)}
			}
		}
	}

	if strings.Contains(message, "API call limit reached") {
		return contracts.LimitError{Message: message}
	}
	if strings.Contains(message, "Request timeout") {
		return contracts.TimeoutError{Message: message}
	}
	if name == "ReferenceError" || strings.Contains(message, "is not defined") {
		return contracts.ReferenceError{Message: message}
	}
	if name == "SyntaxError" || name == "TypeError" || name == "Error" {
		return contracts.ValidationError{Message: message, Details: raw}
	}
	return contracts.ValidationError{Message: message, Details: raw}
}

func normalizeJSError(execCtx context.Context, err error) error {
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return contracts.TimeoutError{Message: "execution timed out", Err: execCtx.Err()}
	}
	if errors.Is(execCtx.Err(), context.Canceled) {
		return contracts.InternalError{Message: "execution canceled", Err: execCtx.Err()}
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "ReferenceError"):
		return contracts.ReferenceError{Message: cleanJSMessage(message)}
	case strings.Contains(message, "SyntaxError"):
		return contracts.ValidationError{Message: cleanJSMessage(message), Err: err}
	default:
		return contracts.ValidationError{Message: cleanJSMessage(message), Err: err}
	}
}

func cleanJSMessage(message string) string {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) == 0 {
		return message
	}
	return strings.TrimSpace(lines[0])
}

func serializedSize(value any) (int64, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func numberToInt(v any) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func stringOr(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func humanBytes(v int64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%dB", v)
	}
	if v < unit*unit {
		return fmt.Sprintf("%.1fKB", float64(v)/unit)
	}
	return fmt.Sprintf("%.1fMB", float64(v)/(unit*unit))
}

func anyToStrings(value any) []string {
	raw, ok := value.([]any)
	if !ok || len(raw) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func normalizePanic(execCtx context.Context, recovered any) error {
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return contracts.TimeoutError{Message: "execution timed out", Err: execCtx.Err()}
	}
	if errors.Is(execCtx.Err(), context.Canceled) {
		return contracts.InternalError{Message: "execution canceled", Err: execCtx.Err()}
	}
	if err, ok := recovered.(error); ok {
		return contracts.InternalError{Message: cleanJSMessage(err.Error()), Err: err}
	}
	return contracts.InternalError{Message: fmt.Sprint(recovered)}
}
