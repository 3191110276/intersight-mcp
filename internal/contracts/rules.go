package contracts

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type RuleCatalog struct {
	Metadata ArtifactSourceMetadata `json:"metadata"`
	Methods  map[string]MethodRules `json:"methods"`
}

type MethodRules struct {
	SDKMethod   string         `json:"sdkMethod"`
	OperationID string         `json:"operationId"`
	Resource    string         `json:"resource"`
	Rules       []SemanticRule `json:"rules,omitempty"`
}

type SemanticRule struct {
	Kind        string         `json:"kind"`
	Description string         `json:"description,omitempty"`
	When        *RuleCondition `json:"when,omitempty"`
	Require     []FieldRule    `json:"require,omitempty"`
	RequireAny  []FieldRule    `json:"requireAny,omitempty"`
	Forbid      []string       `json:"forbid,omitempty"`
	Minimum     []MinimumRule  `json:"minimum,omitempty"`
}

type RuleCondition struct {
	Field  string `json:"field"`
	Equals any    `json:"equals,omitempty"`
	In     []any  `json:"in,omitempty"`
}

type FieldRule struct {
	Field    string `json:"field"`
	Target   string `json:"target,omitempty"`
	MinCount int    `json:"minCount,omitempty"`
}

type MinimumRule struct {
	Field string  `json:"field"`
	Value float64 `json:"value"`
}

func BuildRuleCatalog(spec NormalizedSpec, catalog SDKCatalog) (RuleCatalog, error) {
	rules := RuleCatalog{
		Metadata: spec.Metadata,
		Methods:  map[string]MethodRules{},
	}

	for _, entry := range defaultRuleTemplates() {
		method, ok := catalog.Methods[entry.SDKMethod]
		if !ok {
			continue
		}
		filteredRules := make([]SemanticRule, 0, len(entry.Rules))
		for _, rule := range entry.Rules {
			if strings.TrimSpace(rule.Kind) == "required" {
				continue
			}
			filteredRules = append(filteredRules, rule)
		}
		rules.Methods[entry.SDKMethod] = MethodRules{
			SDKMethod:   entry.SDKMethod,
			OperationID: method.Descriptor.OperationID,
			Resource:    entry.Resource,
			Rules:       filteredRules,
		}
	}

	return rules, nil
}

func ValidateRuleCatalogAgainstArtifacts(spec NormalizedSpec, catalog SDKCatalog, rules RuleCatalog) error {
	if spec.Metadata != catalog.Metadata || spec.Metadata != rules.Metadata {
		return fmt.Errorf("embedded artifact validation failed: spec, sdk catalog, and rule metadata must share identical source metadata")
	}

	expected, err := BuildRuleCatalog(spec, catalog)
	if err != nil {
		return err
	}
	expected = normalizeRuleCatalog(expected)
	rules = normalizeRuleCatalog(rules)

	for sdkMethod, methodRules := range rules.Methods {
		if methodRules.SDKMethod == "" {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q is missing sdkMethod", sdkMethod)
		}
		method, ok := catalog.Methods[sdkMethod]
		if !ok {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown sdk method", sdkMethod)
		}
		if methodRules.OperationID == "" || methodRules.OperationID != method.Descriptor.OperationID {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q does not match sdk catalog operationId", sdkMethod)
		}
		if methodRules.Resource == "" {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q is missing resource", sdkMethod)
		}
		if _, ok := spec.Schemas[methodRules.Resource]; !ok {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown resource schema %q", sdkMethod, methodRules.Resource)
		}
		if method.Resource != "" && method.Resource != methodRules.Resource {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q resource %q does not match sdk catalog resource %q", sdkMethod, methodRules.Resource, method.Resource)
		}

		_, bodySchema, ok := findSpecOperationForDescriptor(spec, method.Descriptor)
		if !ok {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown operation %q", sdkMethod, method.Descriptor.OperationID)
		}
		if bodySchema == nil {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q targets an operation without an application/json request body", sdkMethod)
		}
		if err := validateMethodRules(spec, sdkMethod, methodRules, bodySchema); err != nil {
			return err
		}
	}

	if reflect.DeepEqual(expected, rules) {
		return nil
	}
	for name := range expected.Methods {
		if _, ok := rules.Methods[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: rule metadata missing method %q", name)
		}
	}
	for name := range rules.Methods {
		if _, ok := expected.Methods[name]; !ok {
			return fmt.Errorf("embedded artifact validation failed: rule metadata contains unknown method %q", name)
		}
		if !reflect.DeepEqual(expected.Methods[name], rules.Methods[name]) {
			return fmt.Errorf("embedded artifact validation failed: rule metadata entry %q does not match generated rules", name)
		}
	}
	return fmt.Errorf("embedded artifact validation failed: rule metadata does not match generated rules")
}

func validateMethodRules(spec NormalizedSpec, sdkMethod string, methodRules MethodRules, bodySchema *NormalizedSchema) error {
	for _, rule := range methodRules.Rules {
		kind := strings.TrimSpace(rule.Kind)
		if kind == "" {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q contains a rule without kind", sdkMethod)
		}
		if kind == "required" {
			return fmt.Errorf("embedded artifact validation failed: rules entry %q uses unsupported rule kind %q", sdkMethod, kind)
		}
		if rule.When != nil {
			if _, ok := schemaAtFieldPath(spec, bodySchema, rule.When.Field); !ok {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown condition field %q", sdkMethod, rule.When.Field)
			}
		}
		for _, field := range rule.Forbid {
			if _, ok := schemaAtFieldPath(spec, bodySchema, field); !ok {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown forbidden field %q", sdkMethod, field)
			}
		}
		for _, requirement := range rule.Require {
			schema, ok := schemaAtFieldPath(spec, bodySchema, requirement.Field)
			if !ok {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown required field %q", sdkMethod, requirement.Field)
			}
			if requirement.Target != "" {
				if _, ok := spec.Schemas[requirement.Target]; !ok {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown relationship target %q", sdkMethod, requirement.Target)
				}
				if err := validateRelationshipTarget(requirement.Target, schema); err != nil {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q field %q %w", sdkMethod, requirement.Field, err)
				}
			}
		}
		for _, requirement := range rule.RequireAny {
			schema, ok := schemaAtFieldPath(spec, bodySchema, requirement.Field)
			if !ok {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown one-of field %q", sdkMethod, requirement.Field)
			}
			if requirement.Target != "" {
				if _, ok := spec.Schemas[requirement.Target]; !ok {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown relationship target %q", sdkMethod, requirement.Target)
				}
				if err := validateRelationshipTarget(requirement.Target, schema); err != nil {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q field %q %w", sdkMethod, requirement.Field, err)
				}
			}
		}
		for _, minimum := range rule.Minimum {
			schema, ok := schemaAtFieldPath(spec, bodySchema, minimum.Field)
			if !ok {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown minimum field %q", sdkMethod, minimum.Field)
			}
			switch schema.Type {
			case "integer", "number":
			default:
				return fmt.Errorf("embedded artifact validation failed: rules entry %q minimum field %q must resolve to a numeric schema", sdkMethod, minimum.Field)
			}
		}
	}
	return nil
}

func validateRelationshipTarget(target string, schema *NormalizedSchema) error {
	if schema == nil {
		return fmt.Errorf("does not resolve to a schema")
	}
	if schema.Items != nil {
		schema = schema.Items
	}
	if schema.RelationshipTarget != "" && schema.RelationshipTarget != target {
		return fmt.Errorf("relationship target %q does not match embedded spec target %q", target, schema.RelationshipTarget)
	}
	if schema.Relationship || strings.HasSuffix(schema.Circular, ".Relationship") {
		return nil
	}
	return nil
}

func schemaAtFieldPath(spec NormalizedSpec, root *NormalizedSchema, fieldPath string) (*NormalizedSchema, bool) {
	current := root
	for _, segment := range strings.Split(strings.TrimSpace(fieldPath), ".") {
		if segment == "" {
			return nil, false
		}
		current = dereferenceSchema(spec, current)
		if current == nil {
			return nil, false
		}
		next, ok := current.Properties[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	return dereferenceSchema(spec, current), current != nil
}

func dereferenceSchema(spec NormalizedSpec, schema *NormalizedSchema) *NormalizedSchema {
	if schema == nil {
		return nil
	}
	for schema.Circular != "" {
		target, ok := spec.Schemas[schema.Circular]
		if !ok {
			return schema
		}
		schema = &target
	}
	return schema
}

func normalizeRuleCatalog(catalog RuleCatalog) RuleCatalog {
	if catalog.Methods == nil {
		catalog.Methods = map[string]MethodRules{}
	}
	for key, method := range catalog.Methods {
		method.Rules = normalizeSemanticRules(method.Rules)
		catalog.Methods[key] = method
	}
	return catalog
}

func normalizeSemanticRules(rules []SemanticRule) []SemanticRule {
	if len(rules) == 0 {
		return nil
	}
	out := append([]SemanticRule(nil), rules...)
	for i := range out {
		out[i].Require = append([]FieldRule(nil), out[i].Require...)
		slices.SortFunc(out[i].Require, func(a, b FieldRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].RequireAny = append([]FieldRule(nil), out[i].RequireAny...)
		slices.SortFunc(out[i].RequireAny, func(a, b FieldRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].Forbid = uniqueSortedStrings(out[i].Forbid)
		out[i].Minimum = append([]MinimumRule(nil), out[i].Minimum...)
		slices.SortFunc(out[i].Minimum, func(a, b MinimumRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		if out[i].When != nil && len(out[i].When.In) > 0 {
			out[i].When.In = append([]any(nil), out[i].When.In...)
		}
	}
	return out
}

type methodRuleTemplate struct {
	SDKMethod string
	Resource  string
	Rules     []SemanticRule
}

func defaultRuleTemplates() []methodRuleTemplate {
	return []methodRuleTemplate{
		{
			SDKMethod: "aaa.retentionPolicy.create",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("RetentionPeriod", ""),
				minimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "aaa.retentionPolicy.post",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("RetentionPeriod", ""),
				minimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "aaa.retentionPolicy.update",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("RetentionPeriod", ""),
				minimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "access.policy.create",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("AddressType", ""),
				requiredFieldRule("ConfigurationType", ""),
				whenRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				whenMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "access.policy.post",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("AddressType", ""),
				requiredFieldRule("ConfigurationType", ""),
				whenRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				whenMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "access.policy.update",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("AddressType", ""),
				requiredFieldRule("ConfigurationType", ""),
				whenRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				whenMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.create",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.post",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.update",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.create",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				forbidFieldRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.post",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				forbidFieldRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.update",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				forbidFieldRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.patch",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				forbidFieldRule("Name"),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.create",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.post",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.update",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.create",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PcieZones", "", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.post",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PcieZones", "", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.update",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PcieZones", "", 1),
			},
		},
		{
			SDKMethod: "vnic.ethIf.create",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				requiredFieldRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				requiredFieldRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				whenRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				whenForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				whenRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				whenForbidRule("MacAddressType", "STATIC", "MacPool"),
				whenInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.post",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				requiredFieldRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				requiredFieldRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				whenRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				whenForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				whenRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				whenForbidRule("MacAddressType", "STATIC", "MacPool"),
				whenInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.update",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				requiredFieldRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				requiredFieldRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				whenRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				whenForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				whenRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				whenForbidRule("MacAddressType", "STATIC", "MacPool"),
				whenInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.patch",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				whenRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				whenForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				whenRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				whenForbidRule("MacAddressType", "STATIC", "MacPool"),
				whenInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.create",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				whenRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				whenForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				whenRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				whenForbidRule("IqnAllocationType", "Static", "IqnPool"),
				whenRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.post",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				whenRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				whenForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				whenRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				whenForbidRule("IqnAllocationType", "Static", "IqnPool"),
				whenRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.update",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				whenRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				whenForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				whenRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				whenForbidRule("IqnAllocationType", "Static", "IqnPool"),
				whenRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.patch",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				whenRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				whenForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				whenRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				whenForbidRule("IqnAllocationType", "Static", "IqnPool"),
				whenRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.create",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				whenRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				whenRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.post",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				whenRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				whenRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.update",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				whenRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				whenRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.patch",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				whenRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				whenRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.create",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				whenRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				whenRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				whenMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.post",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				whenRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				whenRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				whenMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.update",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				whenRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				whenRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				whenMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.patch",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				whenRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				whenRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				whenMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.create",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				requiredFieldRule("VlanSettings.AllowedVlans", ""),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.post",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				requiredFieldRule("VlanSettings.AllowedVlans", ""),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.update",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("VlanSettings", ""),
				requiredFieldRule("VlanSettings.AllowedVlans", ""),
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.patch",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				whenRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.create",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.post",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.update",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.create",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.post",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.update",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.create",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.post",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.update",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.create",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpRootPwd", ""),
				requiredFieldRule("HypervisorAdmin", ""),
				requiredFieldRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.post",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpRootPwd", ""),
				requiredFieldRule("HypervisorAdmin", ""),
				requiredFieldRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.update",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpRootPwd", ""),
				requiredFieldRule("HypervisorAdmin", ""),
				requiredFieldRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.create",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
				minimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.post",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
				minimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.update",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Hostname", ""),
				minimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.create",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpVersion", ""),
				requiredFieldRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.post",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpVersion", ""),
				requiredFieldRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.update",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("HxdpVersion", ""),
				requiredFieldRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.create",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.post",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.update",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.create",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.post",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.update",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.create",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DnsServers", "", 1),
				requiredFieldRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.post",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DnsServers", "", 1),
				requiredFieldRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.update",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DnsServers", "", 1),
				requiredFieldRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.create",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DataCenter", ""),
				requiredFieldRule("Hostname", ""),
				requiredFieldRule("Username", ""),
				requiredFieldRule("Password", ""),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.post",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DataCenter", ""),
				requiredFieldRule("Hostname", ""),
				requiredFieldRule("Username", ""),
				requiredFieldRule("Password", ""),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.update",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("DataCenter", ""),
				requiredFieldRule("Hostname", ""),
				requiredFieldRule("Username", ""),
				requiredFieldRule("Password", ""),
			},
		},
		{
			SDKMethod: "ntp.policy.create",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("Timezone", ""),
				oneOfFieldRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.post",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("Timezone", ""),
				oneOfFieldRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.update",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("Timezone", ""),
				oneOfFieldRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.create",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.post",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.update",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.create",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Schedule", ""),
				requiredFieldRule("Schedule.ExecutionTime", ""),
				requiredFieldRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.post",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Schedule", ""),
				requiredFieldRule("Schedule.ExecutionTime", ""),
				requiredFieldRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.update",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Schedule", ""),
				requiredFieldRule("Schedule.ExecutionTime", ""),
				requiredFieldRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.create",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.post",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.update",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policy.create",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("SenderEmail", ""),
				requiredFieldRule("SmtpPort", ""),
				requiredFieldRule("SmtpRecipients", "", 1),
				requiredFieldRule("SmtpServer", ""),
				requiredFieldRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "smtp.policy.post",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("SenderEmail", ""),
				requiredFieldRule("SmtpPort", ""),
				requiredFieldRule("SmtpRecipients", "", 1),
				requiredFieldRule("SmtpServer", ""),
				requiredFieldRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "smtp.policy.update",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("Enabled", ""),
				requiredFieldRule("SenderEmail", ""),
				requiredFieldRule("SmtpPort", ""),
				requiredFieldRule("SmtpRecipients", "", 1),
				requiredFieldRule("SmtpServer", ""),
				requiredFieldRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "syslog.policy.create",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.post",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.update",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				requiredFieldRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.create",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ScheduleParams", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.post",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ScheduleParams", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.update",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ScheduleParams", "", 1),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.create",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.post",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.update",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.create",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				minimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.post",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				minimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.update",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				minimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.create",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.post",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.update",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.create",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("IpAddress", ""),
				requiredFieldRule("IscsiIpType", ""),
				requiredFieldRule("Port", ""),
				requiredFieldRule("TargetName", ""),
				requiredFieldRule("Lun", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.post",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("IpAddress", ""),
				requiredFieldRule("IscsiIpType", ""),
				requiredFieldRule("Port", ""),
				requiredFieldRule("TargetName", ""),
				requiredFieldRule("Lun", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.update",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				requiredFieldRule("IpAddress", ""),
				requiredFieldRule("IscsiIpType", ""),
				requiredFieldRule("Port", ""),
				requiredFieldRule("TargetName", ""),
				requiredFieldRule("Lun", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.create",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VlanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.post",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VlanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.update",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VlanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.create",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VsanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VsanId", ""),
				requiredFieldRule("WwxnPrefixRange", ""),
				requiredFieldRule("WwxnPrefixRange.StartAddr", ""),
				requiredFieldRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.post",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VsanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VsanId", ""),
				requiredFieldRule("WwxnPrefixRange", ""),
				requiredFieldRule("WwxnPrefixRange.StartAddr", ""),
				requiredFieldRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.update",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				requiredFieldRule("ExtaTraffic", ""),
				requiredFieldRule("ExtaTraffic.Name", ""),
				requiredFieldRule("ExtaTraffic.VsanId", ""),
				requiredFieldRule("ExtbTraffic", ""),
				requiredFieldRule("ExtbTraffic.Name", ""),
				requiredFieldRule("ExtbTraffic.VsanId", ""),
				requiredFieldRule("WwxnPrefixRange", ""),
				requiredFieldRule("WwxnPrefixRange.StartAddr", ""),
				requiredFieldRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "smtp.policyTest.create",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				requiredFieldRule("Policy", "smtp.Policy"),
				requiredFieldRule("Recipients", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policyTest.post",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				requiredFieldRule("Policy", "smtp.Policy"),
				requiredFieldRule("Recipients", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policyTest.update",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				requiredFieldRule("Policy", "smtp.Policy"),
				requiredFieldRule("Recipients", "", 1),
			},
		},
	}
}

func requiredFieldRule(field, target string, minCount ...int) SemanticRule {
	requirement := FieldRule{Field: field, Target: target}
	if len(minCount) > 0 {
		requirement.MinCount = minCount[0]
	}
	return SemanticRule{
		Kind:    "required",
		Require: []FieldRule{requirement},
	}
}

func whenRequireRule(field string, equals any, requirement FieldRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, Equals: equals},
		Require: []FieldRule{requirement},
	}
}

func whenInRequireRule(field string, values []any, requirement FieldRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, In: append([]any(nil), values...)},
		Require: []FieldRule{requirement},
	}
}

func whenForbidRule(field string, equals any, forbidden string) SemanticRule {
	return SemanticRule{
		Kind:   "conditional",
		When:   &RuleCondition{Field: field, Equals: equals},
		Forbid: []string{forbidden},
	}
}

func forbidFieldRule(field string) SemanticRule {
	return SemanticRule{
		Kind:   "forbidden",
		Forbid: []string{field},
	}
}

func minimumRule(minimum MinimumRule) SemanticRule {
	return SemanticRule{
		Kind:    "minimum",
		Minimum: []MinimumRule{minimum},
	}
}

func oneOfFieldRule(fields ...string) SemanticRule {
	requireAny := make([]FieldRule, 0, len(fields))
	for _, field := range fields {
		requireAny = append(requireAny, FieldRule{Field: field})
	}
	return SemanticRule{
		Kind:       "one_of",
		RequireAny: requireAny,
	}
}

func whenMinimumRule(field string, equals any, minimum MinimumRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, Equals: equals},
		Minimum: []MinimumRule{minimum},
	}
}
