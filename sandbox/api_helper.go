package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fastschema/qjs"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type apiBridge struct {
	client         APICaller
	mode           Mode
	perCallTimeout time.Duration
	maxAPICalls    int
	callCount      atomic.Int64
	spec           *dryRunSpecIndex
	sdk            *sdkRuntime
	presentation   *PresentationHint
}

func (b *apiBridge) APICallCount() int {
	return int(b.callCount.Load())
}

func compileOperation(method string, path string, options APIRequestOptions) contracts.OperationDescriptor {
	operation := contracts.NewHTTPOperationDescriptor(method, path)
	operation.QueryParams = stringMapToMultiMap(options.Query)
	operation.Headers = stringMapToMultiMap(options.Headers)
	operation.Body = options.Body
	operation.EndpointURL = strings.TrimSpace(options.EndpointURL)
	return operation
}

func stringifyMap(input any) (map[string]string, error) {
	raw, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", input)
	}
	if len(raw) == 0 {
		return map[string]string{}, nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			out[key] = typed
		case nil:
			out[key] = ""
		default:
			return nil, fmt.Errorf("key %q must be a string, got %T", key, value)
		}
	}
	return out, nil
}

func stringMapToMultiMap(in map[string]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = []string{v}
	}
	return out
}

type logBuffer struct {
	data []byte
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *logBuffer) Lines() []string {
	if len(b.data) == 0 {
		return []string{}
	}

	trimmed := strings.TrimRight(string(b.data), "\n")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func rejectionValue(err error, perCallTimeout time.Duration, maxAPICalls int) (map[string]any, error) {
	var authErr contracts.AuthError
	if errors.As(err, &authErr) {
		return map[string]any{
			"kind":    "auth",
			"message": authErr.Error(),
		}, nil
	}

	var httpErr contracts.HTTPError
	if errors.As(err, &httpErr) {
		return map[string]any{
			"kind":   "http",
			"status": httpErr.Status,
			"body":   httpErr.Body,
		}, nil
	}

	var networkErr contracts.NetworkError
	if errors.As(err, &networkErr) {
		return map[string]any{
			"kind":    "network",
			"message": networkErr.Error(),
		}, nil
	}

	var timeoutErr contracts.TimeoutError
	if errors.As(err, &timeoutErr) {
		return map[string]any{
			"kind":           "timeout",
			"message":        fmt.Sprintf("Request timeout (%ds)", int(perCallTimeout/time.Second)),
			"timeoutSeconds": int(perCallTimeout / time.Second),
		}, nil
	}

	var limitErr contracts.LimitError
	if errors.As(err, &limitErr) {
		return map[string]any{
			"kind":    "limit",
			"message": limitErr.Error(),
			"limit":   maxAPICalls,
		}, nil
	}

	return nil, err
}

func resolvePromise(this *qjs.This, payload any) {
	value, err := jsonToJSValue(this.Context(), payload)
	if err != nil {
		rejectPromise(this, contracts.InternalError{Message: "serialize api.call result", Err: err})
		return
	}
	defer value.Free()

	if err := this.Promise().Resolve(value); err != nil {
		_ = this.Context().ThrowError(err)
	}
}

func rejectPromise(this *qjs.This, payload any) {
	value, err := jsonToJSValue(this.Context(), payload)
	if err != nil {
		_ = this.Context().ThrowError(err)
		return
	}
	defer value.Free()

	if err := this.Promise().Reject(value); err != nil {
		_ = this.Context().ThrowError(err)
	}
}

func jsonToJSValue(ctx *qjs.Context, payload any) (*qjs.Value, error) {
	if payload == nil {
		return ctx.NewNull(), nil
	}

	switch typed := payload.(type) {
	case error:
		payload = map[string]any{"message": typed.Error()}
	case string:
		return ctx.NewString(typed), nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return ctx.ParseJSON(string(data)), nil
}
