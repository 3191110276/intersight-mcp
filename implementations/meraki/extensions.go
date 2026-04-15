package meraki

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/mimaurer/intersight-mcp/implementations/meraki/generated"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

const merakiListAllFollowUpKind = "meraki-list-all"

var (
	customSDKMethodsOnce sync.Once
	customSDKMethods     map[string]providerext.CustomSDKMethod
	customSDKMethodsErr  error
)

func SandboxExtensions() providerext.Extensions {
	methods, err := merakiCustomSDKMethods()
	if err != nil {
		panic(err)
	}
	return providerext.Extensions{
		CustomSDKMethods: methods,
	}
}

func merakiCustomSDKMethods() (map[string]providerext.CustomSDKMethod, error) {
	customSDKMethodsOnce.Do(func() {
		customSDKMethods, customSDKMethodsErr = buildMerakiCustomSDKMethods()
	})
	if customSDKMethodsErr != nil {
		return nil, customSDKMethodsErr
	}
	out := make(map[string]providerext.CustomSDKMethod, len(customSDKMethods))
	for name, method := range customSDKMethods {
		out[name] = method
	}
	return out, nil
}

func buildMerakiCustomSDKMethods() (map[string]providerext.CustomSDKMethod, error) {
	var catalog contracts.SDKCatalog
	if err := json.Unmarshal(generated.SDKCatalogBytes(), &catalog); err != nil {
		return nil, fmt.Errorf("decode Meraki SDK catalog for custom methods: %w", err)
	}

	out := map[string]providerext.CustomSDKMethod{}
	for _, sdkMethod := range catalog.Methods {
		if !eligibleMerakiListAllMethod(sdkMethod) {
			continue
		}

		baseMethod := sdkMethod
		helperName := strings.TrimSuffix(baseMethod.SDKMethod, ".list") + ".listAll"
		out[helperName] = providerext.CustomSDKMethod{
			CompileOperation: func(args map[string]any, mode string, _ bool) (contracts.OperationDescriptor, error) {
				return compileMerakiListAllOperation(baseMethod, args, mode)
			},
		}
	}
	return out, nil
}

func eligibleMerakiListAllMethod(method contracts.SDKMethod) bool {
	if !strings.EqualFold(method.Descriptor.Method, "GET") {
		return false
	}
	if !strings.HasSuffix(method.SDKMethod, ".list") {
		return false
	}
	return slices.Contains(method.QueryParameters, "startingAfter")
}

func compileMerakiListAllOperation(method contracts.SDKMethod, args map[string]any, mode string) (contracts.OperationDescriptor, error) {
	if mode != "query" {
		return contracts.OperationDescriptor{}, contracts.ValidationError{
			Message: fmt.Sprintf("%q is a read-only pagination helper and only runs in query", strings.TrimSuffix(method.SDKMethod, ".list")+".listAll"),
			Details: map[string]any{
				"sdkMethod": strings.TrimSuffix(method.SDKMethod, ".list") + ".listAll",
				"toolMode":  mode,
			},
		}
	}

	if args == nil {
		args = map[string]any{}
	}
	pathArgs, err := decodeMerakiNamedArgs(args["path"], "path")
	if err != nil {
		return contracts.OperationDescriptor{}, err
	}
	if err := validateMerakiAllowedKeys("path", pathArgs, method.PathParameters); err != nil {
		return contracts.OperationDescriptor{}, err
	}
	for _, name := range method.PathParameters {
		if strings.TrimSpace(pathArgs[name]) == "" {
			return contracts.OperationDescriptor{}, contracts.ValidationError{
				Message: fmt.Sprintf("sdk method %q is missing required path arguments: %s", strings.TrimSuffix(method.SDKMethod, ".list")+".listAll", name),
				Details: map[string]any{"sdkMethod": strings.TrimSuffix(method.SDKMethod, ".list") + ".listAll"},
			}
		}
	}

	queryArgs, err := decodeMerakiQueryArgs(args["query"])
	if err != nil {
		return contracts.OperationDescriptor{}, err
	}
	if err := validateMerakiAllowedKeys("query", flattenMerakiMultiMap(queryArgs), method.QueryParameters); err != nil {
		return contracts.OperationDescriptor{}, err
	}

	if _, ok := args["body"]; ok {
		return contracts.OperationDescriptor{}, contracts.ValidationError{
			Message: fmt.Sprintf("sdk method %q does not accept body", strings.TrimSuffix(method.SDKMethod, ".list")+".listAll"),
			Details: map[string]any{"sdkMethod": strings.TrimSuffix(method.SDKMethod, ".list") + ".listAll"},
		}
	}

	operation := method.Descriptor
	operation.PathParams = pathArgs
	operation.QueryParams = queryArgs
	operation.Headers = map[string][]string{}
	operation.Path = operation.PathTemplate
	operation.FollowUpPlan = contracts.FollowUpPlan{
		Kind: merakiListAllFollowUpKind,
	}
	return operation, nil
}

func decodeMerakiNamedArgs(raw any, kind string) (map[string]string, error) {
	if raw == nil {
		return map[string]string{}, nil
	}
	typed, ok := raw.(map[string]any)
	if !ok {
		return nil, contracts.ValidationError{Message: fmt.Sprintf("%s arguments must be an object", kind)}
	}
	out := make(map[string]string, len(typed))
	for key, value := range typed {
		str, ok := value.(string)
		if !ok {
			return nil, contracts.ValidationError{Message: fmt.Sprintf("%s argument %q must be a string", kind, key)}
		}
		out[key] = str
	}
	return out, nil
}

func decodeMerakiQueryArgs(raw any) (map[string][]string, error) {
	if raw == nil {
		return map[string][]string{}, nil
	}
	typed, ok := raw.(map[string]any)
	if !ok {
		return nil, contracts.ValidationError{Message: "query arguments must be an object"}
	}
	out := make(map[string][]string, len(typed))
	for key, value := range typed {
		switch v := value.(type) {
		case string:
			out[key] = []string{v}
		case []any:
			items := make([]string, 0, len(v))
			for _, item := range v {
				str, ok := item.(string)
				if !ok {
					return nil, contracts.ValidationError{Message: fmt.Sprintf("query argument %q array entries must be strings", key)}
				}
				items = append(items, str)
			}
			out[key] = items
		default:
			return nil, contracts.ValidationError{Message: fmt.Sprintf("query argument %q must be a string or string array", key)}
		}
	}
	return out, nil
}

func flattenMerakiMultiMap(in map[string][]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, values := range in {
		if len(values) == 0 {
			out[key] = ""
			continue
		}
		out[key] = values[0]
	}
	return out
}

func validateMerakiAllowedKeys(kind string, values map[string]string, allowed []string) error {
	if len(values) == 0 {
		return nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
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
	slices.Sort(unknown)
	return contracts.ValidationError{
		Message: fmt.Sprintf("unknown %s arguments: %s", kind, strings.Join(unknown, ", ")),
	}
}
