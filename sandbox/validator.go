package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

var pathTemplateParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

func (b *apiBridge) callSDK(execCtx context.Context, sdkMethod string, args map[string]any) (map[string]any, error) {
	if b.sdk == nil {
		return nil, contracts.ReferenceError{Message: "sdk is not available in this runtime"}
	}

	if b.sdk.isCustomSDKMethod(sdkMethod) {
		return b.callCustomSDK(execCtx, sdkMethod, args)
	}

	if b.mode == ModeValidate {
		return b.validateSDKOffline(sdkMethod, args)
	}
	if b.mode == ModeQuery {
		method, err := b.sdk.sdkMethod(sdkMethod)
		if err != nil {
			rejection, rejectErr := rejectionValue(err, b.perCallTimeout, b.maxAPICalls)
			if rejectErr != nil {
				return nil, rejectErr
			}
			return map[string]any{
				"ok":    false,
				"error": rejection,
			}, nil
		}
		if !strings.EqualFold(method.Descriptor.Method, http.MethodGet) {
			return b.validateSDKOffline(sdkMethod, args)
		}
	}

	operation, validationReport, err := b.sdk.compileOperation(sdkMethod, args, b.mode)
	if err != nil {
		rejection, rejectErr := rejectionValue(err, b.perCallTimeout, b.maxAPICalls)
		if rejectErr != nil {
			return nil, rejectErr
		}
		return map[string]any{
			"ok":    false,
			"error": rejection,
		}, nil
	}

	if b.mode == ModeMutate {
		if err := b.preflightMutation(execCtx, sdkMethod, operation); err != nil {
			return nil, err
		}
	}

	result, err := b.executeOperation(execCtx, operation)
	if err != nil {
		rejection, rejectErr := rejectionValue(err, b.perCallTimeout, b.maxAPICalls)
		if rejectErr != nil {
			return nil, rejectErr
		}
		return map[string]any{
			"ok":    false,
			"error": rejection,
		}, nil
	}
	if b.mode == ModeMutate {
		if err := b.verifyMutationResult(execCtx, sdkMethod, operation, result); err != nil {
			return nil, err
		}
	}

	response := map[string]any{
		"ok":    true,
		"value": result,
	}
	if len(validationReport) > 0 {
		response["logs"] = validationReport
	}
	return response, nil
}

func (b *apiBridge) callCustomSDK(execCtx context.Context, sdkMethod string, args map[string]any) (map[string]any, error) {
	operation, err := b.sdk.compileCustomOperation(sdkMethod, args, b.mode, b.enableMetricsApps)
	if err != nil {
		rejection, rejectErr := rejectionValue(err, b.perCallTimeout, b.maxAPICalls)
		if rejectErr != nil {
			return nil, rejectErr
		}
		return map[string]any{
			"ok":    false,
			"error": rejection,
		}, nil
	}

	result, err := b.executeOperation(execCtx, operation)
	if err != nil {
		rejection, rejectErr := rejectionValue(err, b.perCallTimeout, b.maxAPICalls)
		if rejectErr != nil {
			return nil, rejectErr
		}
		return map[string]any{
			"ok":    false,
			"error": rejection,
		}, nil
	}

	if custom := b.sdk.extensions.CustomSDKMethods[strings.TrimSpace(sdkMethod)]; custom.PresentationHint != nil {
		b.presentation = custom.PresentationHint(sdkMethod, args, b.enableMetricsApps)
	}

	return map[string]any{
		"ok":    true,
		"value": result,
	}, nil
}

func (b *apiBridge) executeOperation(execCtx context.Context, operation contracts.OperationDescriptor) (any, error) {
	count := int(b.callCount.Add(1))
	if count > b.maxAPICalls {
		return nil, contracts.LimitError{Message: fmt.Sprintf("API call limit reached (%d/%d)", b.maxAPICalls, b.maxAPICalls)}
	}

	callCtx, cancel := context.WithTimeout(execCtx, b.perCallTimeout)
	defer cancel()

	return b.client.Do(callCtx, operation)
}

func (r *sdkRuntime) sdkMethod(sdkMethod string) (contracts.SDKMethod, error) {
	method, ok := r.catalog.Methods[sdkMethod]
	if !ok {
		return contracts.SDKMethod{}, contracts.ValidationError{
			Message: fmt.Sprintf("unknown sdk method %q", sdkMethod),
			Details: map[string]any{"sdkMethod": sdkMethod},
		}
	}
	return method, nil
}

func (r *sdkRuntime) compileCustomOperation(sdkMethod string, args map[string]any, mode Mode, enableMetricsApps bool) (contracts.OperationDescriptor, error) {
	method, ok := r.extensions.CustomSDKMethods[strings.TrimSpace(sdkMethod)]
	if !ok || method.CompileOperation == nil {
		return contracts.OperationDescriptor{}, contracts.ValidationError{
			Message: fmt.Sprintf("unknown custom sdk method %q", sdkMethod),
			Details: map[string]any{"sdkMethod": sdkMethod},
		}
	}
	return method.CompileOperation(args, string(mode), enableMetricsApps)
}

func (r *sdkRuntime) isCustomSDKMethod(sdkMethod string) bool {
	_, ok := r.extensions.CustomSDKMethods[strings.TrimSpace(sdkMethod)]
	return ok
}

func (b *apiBridge) validateSDKOffline(sdkMethod string, args map[string]any) (map[string]any, error) {
	operation, _, err := b.sdk.compileOperation(sdkMethod, args, ModeValidate)
	if err == nil {
		return map[string]any{
			"ok":    true,
			"value": b.sdk.validationSuccessReport(sdkMethod, operation),
		}, nil
	}

	var validationErr contracts.ValidationError
	if errorsAsValidation := errorAsValidation(err, &validationErr); errorsAsValidation {
		if shouldSurfaceValidateModeError(validationErr) {
			return nil, validationErr
		}
		return map[string]any{
			"ok":    true,
			"value": b.sdk.validationFailureReport(sdkMethod, validationErr),
		}, nil
	}

	return nil, err
}

func (r *sdkRuntime) compileOperation(sdkMethod string, args map[string]any, mode Mode) (contracts.OperationDescriptor, []string, error) {
	method, err := r.sdkMethod(sdkMethod)
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}

	if err := guardSDKMethod(mode, method.Descriptor.Method, sdkMethod); err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}

	specOp, schema, err := r.lookupOperation(method)
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}

	normalizedArgs, err := validateSDKArgs(args, method)
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}

	operation := method.Descriptor
	operation.PathParams = map[string]string{}
	operation.QueryParams = map[string][]string{}
	operation.Headers = map[string][]string{}
	var validationReport []string

	pathArgs, err := decodeNamedStringMap(normalizedArgs["path"], "path")
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	if err := validatePathArgs(method, pathArgs); err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	operation.PathParams = pathArgs

	queryArgs, err := decodeQueryArgMap(normalizedArgs["query"])
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	if err := validateAllowedMultiKeys("query", queryArgs, method.QueryParameters); err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	operation.QueryParams = queryArgs

	headerArgs, err := compileHeaderArgs(normalizedArgs, method, specOp)
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	operation.Headers = headerArgs

	body, hasBody := normalizedArgs["body"]
	if hasBody {
		if schema == nil {
			return contracts.OperationDescriptor{}, nil, newSDKContractValidationError(sdkMethod, fmt.Sprintf("sdk method %q does not accept body", sdkMethod), validationIssue{
				Type:      "sdk_contract",
				Source:    validationSourceSDKContract,
				Message:   fmt.Sprintf("sdk method %q does not accept body", sdkMethod),
				SDKMethod: sdkMethod,
			}, defaultValidationLayers(false))
		}
		body = normalizeValueForSchema(r.specIndex, schema, body, &schemaValidationState{visiting: map[string]int{}})
		schemaIssues := validateRequestBodyAgainstSchema(r.specIndex, schema, body)
		ruleIssues := r.validateSemanticRules(sdkMethod, body)
		if len(schemaIssues) > 0 || len(ruleIssues) > 0 {
			layers := defaultValidationLayers(true)
			layers[1].Passed = len(schemaIssues) == 0
			layers[2].Passed = len(ruleIssues) == 0
			issues := append([]dryRunValidationError{}, schemaIssues...)
			issues = append(issues, ruleIssues...)
			validationErr := contracts.ValidationError{
				Message: fmt.Sprintf("sdk method %q request body failed local validation", sdkMethod),
				Details: map[string]any{
					"sdkMethod": sdkMethod,
					"issues":    issues,
					"layers":    layers,
				},
			}
			if mode != ModeMutate {
				return contracts.OperationDescriptor{}, nil, validationErr
			}
			operation.Body = body
			validationReport = validationWarningLogs(sdkMethod, validationErr)
		}
		operation.Body = body
	} else if method.RequestBodyRequired {
		message := fmt.Sprintf("sdk method %q requires body", sdkMethod)
		return contracts.OperationDescriptor{}, nil, newSDKContractValidationError(sdkMethod, message, validationIssue{
			Type:      "sdk_contract",
			Source:    validationSourceSDKContract,
			Message:   message,
			SDKMethod: sdkMethod,
		}, defaultValidationLayers(false))
	}

	resolvedPath, err := resolveTemplatePath(operation.PathTemplate, operation.PathParams)
	if err != nil {
		return contracts.OperationDescriptor{}, nil, err
	}
	operation.Path = resolvedPath

	if operation.Body != nil {
		if ok, message := validatePathBodyMoid(operation.Path, operation.Body); !ok {
			return contracts.OperationDescriptor{}, nil, newSDKContractValidationError(sdkMethod, message, validationIssue{
				Path:      "body.Moid",
				Type:      "sdk_contract",
				Source:    validationSourceSDKContract,
				Message:   message,
				SDKMethod: sdkMethod,
			}, defaultValidationLayers(true))
		}
	}

	return operation, validationReport, nil
}

func validationWarningLogs(sdkMethod string, err contracts.ValidationError) []string {
	report := map[string]any{
		"warning":   "local validation failed but mutate continued; the server response is authoritative",
		"sdkMethod": sdkMethod,
	}
	if details, ok := err.Details.(map[string]any); ok {
		report["issues"] = normalizeValidationIssues(details["issues"], sdkMethod)
		report["layers"] = normalizeValidationLayers(details["layers"])
	}

	lines := []string{
		fmt.Sprintf("Local validation warning for %s: mutate continued and deferred blocking to the server.", sdkMethod),
	}
	if encoded, marshalErr := json.Marshal(report); marshalErr == nil {
		lines = append(lines, string(encoded))
	} else {
		lines = append(lines, err.Error())
	}
	return lines
}

func (r *sdkRuntime) validationSuccessReport(sdkMethod string, operation contracts.OperationDescriptor) map[string]any {
	return map[string]any{
		"valid":     true,
		"sdkMethod": sdkMethod,
		"operation": map[string]any{
			"operationId":  operation.OperationID,
			"method":       operation.Method,
			"path":         operation.Path,
			"pathTemplate": operation.PathTemplate,
		},
		"issues": []validationIssue{},
		"layers": defaultValidationLayers(operation.Body != nil),
	}
}

func (r *sdkRuntime) validationFailureReport(sdkMethod string, err contracts.ValidationError) map[string]any {
	issues := []validationIssue{{
		Type:      "sdk_contract",
		Source:    validationSourceSDKContract,
		Message:   err.Error(),
		SDKMethod: sdkMethod,
	}}
	layers := defaultValidationLayers(false)
	layers[0].Passed = false

	if details, ok := err.Details.(map[string]any); ok {
		if normalized := normalizeValidationIssues(details["issues"], sdkMethod); len(normalized) > 0 {
			issues = normalized
		}
		if normalizedLayers := normalizeValidationLayers(details["layers"]); len(normalizedLayers) > 0 {
			layers = normalizedLayers
		}
	}

	return map[string]any{
		"valid":     false,
		"sdkMethod": sdkMethod,
		"issues":    issues,
		"layers":    layers,
	}
}

func normalizeValidationIssues(raw any, sdkMethod string) []validationIssue {
	switch typed := raw.(type) {
	case []validationIssue:
		out := append([]validationIssue(nil), typed...)
		for i := range out {
			out[i].SDKMethod = firstNonEmpty(out[i].SDKMethod, sdkMethod)
		}
		return out
	case []dryRunValidationError:
		out := make([]validationIssue, 0, len(typed))
		for _, err := range typed {
			out = append(out, validationIssue{
				Path:      err.Path,
				Type:      err.Type,
				Source:    err.Source,
				Message:   err.Message,
				Rule:      err.Rule,
				Condition: err.Condition,
				Expected:  err.Expected,
				Actual:    err.Actual,
				SDKMethod: sdkMethod,
			})
		}
		return out
	case []any:
		out := make([]validationIssue, 0, len(typed))
		for _, entry := range typed {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, validationIssue{
				Path:      stringField(item, "path"),
				Type:      stringField(item, "type"),
				Source:    stringField(item, "source"),
				Message:   stringField(item, "message"),
				Rule:      stringField(item, "rule"),
				Condition: stringField(item, "condition"),
				Expected:  item["expected"],
				Actual:    item["actual"],
				SDKMethod: firstNonEmpty(stringField(item, "sdkMethod"), sdkMethod),
			})
		}
		return out
	default:
		return nil
	}
}

func normalizeValidationLayers(raw any) []validationLayer {
	switch typed := raw.(type) {
	case []validationLayer:
		return append([]validationLayer(nil), typed...)
	case []any:
		out := make([]validationLayer, 0, len(typed))
		for _, entry := range typed {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, validationLayer{
				Name:   stringField(item, "name"),
				Source: stringField(item, "source"),
				Ran:    boolField(item, "ran"),
				Passed: boolField(item, "passed"),
			})
		}
		return out
	default:
		return nil
	}
}

func shouldSurfaceValidateModeError(err contracts.ValidationError) bool {
	details, ok := err.Details.(map[string]any)
	if !ok {
		return false
	}
	return details["toolMode"] == string(ModeValidate)
}

func (r *sdkRuntime) lookupOperation(method contracts.SDKMethod) (contracts.NormalizedOperation, *dryRunSchema, error) {
	methods := r.spec.Paths[method.Descriptor.PathTemplate]
	op, ok := methods[strings.ToLower(method.Descriptor.Method)]
	if !ok {
		return contracts.NormalizedOperation{}, nil, contracts.InternalError{
			Message: fmt.Sprintf("sdk catalog method %q does not map to an embedded spec operation", method.SDKMethod),
		}
	}
	return op, r.specIndex.requestSchema(method.Descriptor.Method, method.Descriptor.PathTemplate), nil
}

func validateSDKArgs(args map[string]any, method contracts.SDKMethod) (map[string]any, error) {
	if args == nil {
		return map[string]any{}, nil
	}

	allowed := map[string]struct{}{
		"path":  {},
		"query": {},
		"body":  {},
	}
	for _, name := range method.HeaderParameters {
		allowed[name] = struct{}{}
	}

	var unknown []string
	for key := range args {
		if _, ok := allowed[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		message := fmt.Sprintf("sdk method %q received unknown arguments: %s", method.SDKMethod, strings.Join(unknown, ", "))
		return nil, newSDKContractValidationError(method.SDKMethod, message, validationIssue{
			Type:      "unknown_field",
			Source:    validationSourceSDKContract,
			Message:   message,
			SDKMethod: method.SDKMethod,
			Actual:    unknown,
		}, defaultValidationLayers(false))
	}

	return args, nil
}

func validatePathArgs(method contracts.SDKMethod, pathArgs map[string]string) error {
	if err := validateAllowedKeys("path", pathArgs, method.PathParameters); err != nil {
		return err
	}

	var missing []string
	for _, name := range method.PathParameters {
		if strings.TrimSpace(pathArgs[name]) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		message := fmt.Sprintf("sdk method %q is missing required path arguments: %s", method.SDKMethod, strings.Join(missing, ", "))
		return newSDKContractValidationError(method.SDKMethod, message, validationIssue{
			Type:      "required",
			Source:    validationSourceSDKContract,
			Message:   message,
			SDKMethod: method.SDKMethod,
			Actual:    missing,
		}, defaultValidationLayers(false))
	}
	return nil
}

func validateAllowedKeys(kind string, values map[string]string, allowed []string) error {
	if len(values) == 0 {
		return nil
	}

	allowedSet := map[string]struct{}{}
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}

	var unknown []string
	for name := range values {
		if _, ok := allowedSet[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	message := fmt.Sprintf("unknown %s arguments: %s", kind, strings.Join(unknown, ", "))
	return contracts.ValidationError{
		Message: message,
		Details: map[string]any{
			"issues": []validationIssue{{
				Type:    "unknown_field",
				Source:  validationSourceSDKContract,
				Message: message,
				Actual:  unknown,
			}},
			"layers": sdkContractFailureLayers(),
		},
	}
}

func validateAllowedMultiKeys(kind string, values map[string][]string, allowed []string) error {
	if len(values) == 0 {
		return nil
	}

	flattened := make(map[string]string, len(values))
	for name, entries := range values {
		if len(entries) == 0 {
			flattened[name] = ""
			continue
		}
		flattened[name] = entries[0]
	}
	return validateAllowedKeys(kind, flattened, allowed)
}

func compileHeaderArgs(args map[string]any, method contracts.SDKMethod, specOp contracts.NormalizedOperation) (map[string][]string, error) {
	headers := map[string][]string{}
	required := map[string]bool{}
	for _, param := range specOp.Parameters {
		if strings.EqualFold(param.In, "header") {
			required[param.Name] = param.Required
		}
	}

	for _, name := range method.HeaderParameters {
		raw, ok := args[name]
		if !ok {
			if required[name] {
				message := fmt.Sprintf("sdk method %q is missing required header argument %q", method.SDKMethod, name)
				return nil, newSDKContractValidationError(method.SDKMethod, message, validationIssue{
					Path:      name,
					Type:      "required",
					Source:    validationSourceSDKContract,
					Message:   message,
					SDKMethod: method.SDKMethod,
				}, defaultValidationLayers(false))
			}
			continue
		}
		value, ok := raw.(string)
		if !ok {
			message := fmt.Sprintf("sdk header argument %q must be a string", name)
			return nil, newSDKContractValidationError(method.SDKMethod, message, validationIssue{
				Path:      name,
				Type:      "type_mismatch",
				Source:    validationSourceSDKContract,
				Message:   message,
				Expected:  "string",
				Actual:    fmt.Sprintf("%T", raw),
				SDKMethod: method.SDKMethod,
			}, defaultValidationLayers(false))
		}
		headers[name] = []string{value}
	}

	return headers, nil
}

func decodeNamedStringMap(raw any, kind string) (map[string]string, error) {
	if raw == nil {
		return map[string]string{}, nil
	}
	values, err := stringifyMap(raw)
	if err != nil {
		message := fmt.Sprintf("sdk %s arguments must be a string map", kind)
		return nil, contracts.ValidationError{
			Message: message,
			Err:     err,
			Details: map[string]any{
				"issues": []validationIssue{{
					Type:     "type_mismatch",
					Source:   validationSourceSDKContract,
					Message:  message,
					Expected: "string map",
					Actual:   fmt.Sprintf("%T", raw),
				}},
				"layers": sdkContractFailureLayers(),
			},
		}
	}
	return values, nil
}

func decodeQueryArgMap(raw any) (map[string][]string, error) {
	if raw == nil {
		return map[string][]string{}, nil
	}

	source, ok := raw.(map[string]any)
	if !ok {
		return nil, contracts.ValidationError{
			Message: "sdk query arguments must be an object",
			Err:     fmt.Errorf("expected object, got %T", raw),
			Details: map[string]any{
				"issues": []validationIssue{{
					Type:     "type_mismatch",
					Source:   validationSourceSDKContract,
					Message:  "sdk query arguments must be an object",
					Expected: "object",
					Actual:   fmt.Sprintf("%T", raw),
				}},
				"layers": sdkContractFailureLayers(),
			},
		}
	}
	if len(source) == 0 {
		return map[string][]string{}, nil
	}

	out := make(map[string][]string, len(source))
	for key, value := range source {
		encoded, err := stringifyQueryValue(value)
		if err != nil {
			return nil, contracts.ValidationError{
				Message: "sdk query arguments must contain only string, number, boolean, null, or array values",
				Err:     fmt.Errorf("key %q: %w", key, err),
				Details: map[string]any{
					"issues": []validationIssue{{
						Path:    key,
						Type:    "type_mismatch",
						Source:  validationSourceSDKContract,
						Message: "sdk query arguments must contain only string, number, boolean, null, or array values",
					}},
					"layers": sdkContractFailureLayers(),
				},
			}
		}
		out[key] = encoded
	}
	return out, nil
}

func stringifyQueryValue(value any) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return []string{""}, nil
	case string:
		return []string{typed}, nil
	case bool:
		if typed {
			return []string{"true"}, nil
		}
		return []string{"false"}, nil
	case int:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case int8:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case int16:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case int32:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case int64:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case uint:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case uint8:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case uint16:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case uint32:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case uint64:
		return []string{fmt.Sprintf("%d", typed)}, nil
	case float64:
		return []string{fmt.Sprintf("%v", typed)}, nil
	case float32:
		return []string{fmt.Sprintf("%v", typed)}, nil
	case []any:
		values := make([]string, 0, len(typed))
		for _, entry := range typed {
			encoded, err := stringifyQueryValue(entry)
			if err != nil {
				return nil, err
			}
			if len(encoded) != 1 {
				return nil, fmt.Errorf("nested arrays are not supported")
			}
			values = append(values, encoded[0])
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", value)
	}
}

func guardSDKMethod(mode Mode, method, sdkMethod string) error {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	switch mode {
	case ModeQuery:
		if normalizedMethod == http.MethodGet {
			return nil
		}
		return contracts.ValidationError{
			Message: fmt.Sprintf("query only allows read-shaped sdk calls; %q is %s and must run in mutate", sdkMethod, normalizedMethod),
			Details: map[string]any{
				"sdkMethod": sdkMethod,
				"method":    normalizedMethod,
			},
		}
	case ModeValidate:
		if normalizedMethod != http.MethodGet {
			return nil
		}
		return contracts.ValidationError{
			Message: fmt.Sprintf("offline validation only accepts write-shaped sdk calls; %q is %s and should run as a normal query", sdkMethod, normalizedMethod),
			Details: map[string]any{
				"sdkMethod": sdkMethod,
				"method":    normalizedMethod,
				"toolMode":  string(mode),
			},
		}
	default:
		return nil
	}
}

func errorAsValidation(err error, target *contracts.ValidationError) bool {
	return target != nil && errors.As(err, target)
}

func resolveTemplatePath(template string, params map[string]string) (string, error) {
	missing := []string{}
	resolved := pathTemplateParamPattern.ReplaceAllStringFunc(template, func(segment string) string {
		matches := pathTemplateParamPattern.FindStringSubmatch(segment)
		if len(matches) != 2 {
			return segment
		}
		name := matches[1]
		value := strings.TrimSpace(params[name])
		if value == "" {
			missing = append(missing, name)
			return segment
		}
		return url.PathEscape(value)
	})
	if len(missing) > 0 {
		message := fmt.Sprintf("missing required path params: %s", strings.Join(missing, ", "))
		return "", contracts.ValidationError{
			Message: message,
			Details: map[string]any{
				"issues": []validationIssue{{
					Type:    "required",
					Source:  validationSourceSDKContract,
					Message: message,
					Actual:  missing,
				}},
				"layers": sdkContractFailureLayers(),
			},
		}
	}
	return resolved, nil
}

func newSDKContractValidationError(sdkMethod, message string, issue validationIssue, layers []validationLayer) contracts.ValidationError {
	issue.SDKMethod = firstNonEmpty(issue.SDKMethod, sdkMethod)
	if issue.Source == "" {
		issue.Source = validationSourceSDKContract
	}
	if issue.Type == "" {
		issue.Type = "sdk_contract"
	}
	return contracts.ValidationError{
		Message: message,
		Details: map[string]any{
			"sdkMethod": sdkMethod,
			"issues":    []validationIssue{issue},
			"layers":    sdkContractFailureLayersWith(layers),
		},
	}
}

func sdkContractFailureLayers() []validationLayer {
	return sdkContractFailureLayersWith(defaultValidationLayers(false))
}

func sdkContractFailureLayersWith(layers []validationLayer) []validationLayer {
	if len(layers) == 0 {
		layers = defaultValidationLayers(false)
	}
	out := append([]validationLayer(nil), layers...)
	out[0].Passed = false
	return out
}

func stringField(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func boolField(item map[string]any, key string) bool {
	value, _ := item[key].(bool)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
