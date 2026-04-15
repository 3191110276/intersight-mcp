package intersight

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
	"github.com/mimaurer/intersight-mcp/internal/providerext"
)

const telemetryQuerySDKMethod = "telemetry.query"

func SandboxExtensions() providerext.Extensions {
	return providerext.Extensions{
		CustomSDKMethods: map[string]providerext.CustomSDKMethod{
			telemetryQuerySDKMethod: {
				CompileOperation: compileTelemetryQueryOperation,
			},
		},
		AutofillDiscriminators: true,
		RelationshipBehavior: &providerext.RelationshipBehavior{
			RejectSelector:             true,
			SelectorMessage:            "Selector-only relationship payloads are not accepted for writes; provide a Moid reference or typed MoRef.",
			MoidField:                  "Moid",
			ClassIDField:               "ClassId",
			ObjectTypeField:            "ObjectType",
			DefaultClassID:             "mo.MoRef",
			RequiredClassID:            "mo.MoRef",
			AutofillTargetObjectType:   true,
			AllowMoidRefWriteForm:      "moidRef",
			AllowTypedMoRefWriteForm:   "typedMoRef",
			RelationshipRuleName:       "relationship",
			MissingMoidMessage:         "Relationship Moid is required.",
			InvalidPayloadShapeMessage: "Relationship payload shape is not accepted for writes.",
		},
		RelationshipPathResolver: intersightObjectTypeToPath,
		DeleteDependencyRules: map[string][]providerext.DeleteDependencyRule{
			"/api/v1/vnic/LanConnectivityPolicies": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "LanConnectivityPolicy", Label: "attached Ethernet interfaces"},
			},
			"/api/v1/macpool/Pools": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "MacPool", Label: "Ethernet interfaces using this MAC pool"},
			},
			"/api/v1/fabric/EthNetworkGroupPolicies": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "FabricEthNetworkGroupPolicy", Label: "Ethernet interfaces using this Ethernet Network Group Policy"},
			},
			"/api/v1/vnic/EthNetworkPolicies": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthNetworkPolicy", Label: "Ethernet interfaces using this Ethernet Network Policy"},
			},
			"/api/v1/vnic/EthQosPolicies": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthQosPolicy", Label: "Ethernet interfaces using this Ethernet QoS Policy"},
			},
			"/api/v1/vnic/EthAdapterPolicies": {
				{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthAdapterPolicy", Label: "Ethernet interfaces using this Ethernet Adapter Policy"},
			},
		},
	}
}

func compileTelemetryQueryOperation(args map[string]any, mode string, enableMetricsApps bool) (contracts.OperationDescriptor, error) {
	if mode != "query" {
		return contracts.OperationDescriptor{}, contracts.ValidationError{
			Message: fmt.Sprintf("%q is a read-only telemetry query and only runs in query", telemetryQuerySDKMethod),
			Details: map[string]any{
				"sdkMethod": telemetryQuerySDKMethod,
				"method":    http.MethodPost,
				"toolMode":  mode,
			},
		}
	}
	if args == nil {
		args = map[string]any{}
	}

	allowed := map[string]struct{}{
		"dataSource":       {},
		"dimensions":       {},
		"virtualColumns":   {},
		"limitSpec":        {},
		"having":           {},
		"granularity":      {},
		"filter":           {},
		"aggregations":     {},
		"postAggregations": {},
		"intervals":        {},
		"subtotalsSpec":    {},
		"context":          {},
		"render":           {},
	}
	var unknown []string
	for key := range args {
		if _, ok := allowed[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return contracts.OperationDescriptor{}, contracts.ValidationError{
			Message: fmt.Sprintf("sdk method %q received unknown arguments: %s", telemetryQuerySDKMethod, strings.Join(unknown, ", ")),
			Details: map[string]any{"sdkMethod": telemetryQuerySDKMethod},
		}
	}

	dataSource, ok := args["dataSource"]
	if !ok || dataSource == nil {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q requires dataSource", telemetryQuerySDKMethod)}
	}
	if _, ok := dataSource.(string); !ok {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q dataSource must be a string", telemetryQuerySDKMethod)}
	}
	dimensions, ok := args["dimensions"]
	if !ok || dimensions == nil {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q requires dimensions", telemetryQuerySDKMethod)}
	}
	if !isArrayLike(dimensions) {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q dimensions must be an array", telemetryQuerySDKMethod)}
	}
	granularity, ok := args["granularity"]
	if !ok || granularity == nil {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q requires granularity", telemetryQuerySDKMethod)}
	}
	intervals, ok := args["intervals"]
	if !ok || intervals == nil {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q requires intervals", telemetryQuerySDKMethod)}
	}
	if !isArrayLike(intervals) {
		return contracts.OperationDescriptor{}, contracts.ValidationError{Message: fmt.Sprintf("sdk method %q intervals must be an array", telemetryQuerySDKMethod)}
	}
	if _, err := requireTelemetryRenderMode(args, enableMetricsApps); err != nil {
		return contracts.OperationDescriptor{}, err
	}

	bodyObject := map[string]any{
		"queryType":   "groupBy",
		"dataSource":  dataSource,
		"dimensions":  dimensions,
		"granularity": granularity,
		"intervals":   intervals,
	}
	for _, key := range []string{"virtualColumns", "limitSpec", "having", "filter", "aggregations", "postAggregations", "subtotalsSpec", "context"} {
		if value, exists := args[key]; exists {
			bodyObject[key] = value
		}
	}

	operation := contracts.NewHTTPOperationDescriptor(http.MethodPost, "/api/v1/telemetry/TimeSeries")
	operation.OperationID = "CustomTelemetryQuery"
	operation.Body = bodyObject
	return operation, nil
}

func intersightObjectTypeToPath(objectType string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(objectType), ".")
	if len(parts) != 2 {
		return "", false
	}
	return "/api/v1/" + parts[0] + "/" + pluralizeType(parts[1]), true
}

func pluralizeType(name string) string {
	switch {
	case strings.HasSuffix(name, "y") && len(name) > 1 && !strings.ContainsRune("aeiouAEIOU", rune(name[len(name)-2])):
		return name[:len(name)-1] + "ies"
	case strings.HasSuffix(name, "s"):
		return name + "es"
	default:
		return name + "s"
	}
}

func isArrayLike(value any) bool {
	switch value.(type) {
	case []any, []string, []int, []int64, []float64, []bool, []map[string]any:
		return true
	default:
		return false
	}
}

const telemetryRenderOff = "off"

func requireTelemetryRenderMode(args map[string]any, enableMetricsApps bool) (string, error) {
	render, ok := telemetryRenderMode(args, enableMetricsApps)
	if ok {
		return render, nil
	}

	return "", contracts.ValidationError{
		Message: fmt.Sprintf("sdk method %q render must be one of: %s", telemetryQuerySDKMethod, telemetryRenderOff),
		Details: map[string]any{"sdkMethod": telemetryQuerySDKMethod},
	}
}

func telemetryRenderMode(args map[string]any, enableMetricsApps bool) (string, bool) {
	if args == nil {
		return telemetryRenderOff, true
	}
	value, exists := args["render"]
	if !exists || value == nil {
		return telemetryRenderOff, true
	}
	render, ok := value.(string)
	if !ok {
		return "", false
	}
	switch strings.TrimSpace(render) {
	case telemetryRenderOff:
		return telemetryRenderOff, true
	default:
		return "", false
	}
}
