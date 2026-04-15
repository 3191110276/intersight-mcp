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

type RuleTemplate struct {
	SDKMethod string
	Resource  string
	Rules     []SemanticRule
}

func BuildRuleCatalog(spec NormalizedSpec, catalog SDKCatalog, templates []RuleTemplate) (RuleCatalog, error) {
	rules := RuleCatalog{
		Metadata: spec.Metadata,
		Methods:  map[string]MethodRules{},
	}

	for _, entry := range templates {
		method, ok := resolveRuleTemplateMethod(catalog, entry)
		if !ok {
			continue
		}
		_, bodySchema, _ := findSpecOperationForDescriptor(spec, method.Descriptor)
		filteredRules := make([]SemanticRule, 0, len(entry.Rules))
		for _, rule := range entry.Rules {
			if strings.TrimSpace(rule.Kind) == "required" && shouldOmitRequiredRule(spec, bodySchema, rule) {
				continue
			}
			filteredRules = append(filteredRules, rule)
		}
		rules.Methods[method.SDKMethod] = MethodRules{
			SDKMethod:   method.SDKMethod,
			OperationID: method.Descriptor.OperationID,
			Resource:    entry.Resource,
			Rules:       filteredRules,
		}
	}

	return rules, nil
}

func resolveRuleTemplateMethod(catalog SDKCatalog, entry RuleTemplate) (SDKMethod, bool) {
	if method, ok := catalog.Methods[entry.SDKMethod]; ok {
		return method, true
	}

	verb := sdkMethodVerb(entry.SDKMethod)
	if verb == "" || strings.TrimSpace(entry.Resource) == "" {
		return SDKMethod{}, false
	}

	var match SDKMethod
	for _, method := range catalog.Methods {
		if method.Resource != entry.Resource || sdkMethodVerb(method.SDKMethod) != verb {
			continue
		}
		if match.SDKMethod != "" {
			return SDKMethod{}, false
		}
		match = method
	}
	if match.SDKMethod == "" {
		return SDKMethod{}, false
	}
	return match, true
}

func sdkMethodVerb(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	idx := strings.LastIndexByte(name, '.')
	if idx < 0 || idx == len(name)-1 {
		return ""
	}
	return name[idx+1:]
}

func ValidateRuleCatalogAgainstArtifacts(spec NormalizedSpec, catalog SDKCatalog, rules RuleCatalog, templates []RuleTemplate) error {
	if spec.Metadata != catalog.Metadata || spec.Metadata != rules.Metadata {
		return fmt.Errorf("embedded artifact validation failed: spec, sdk catalog, and rule metadata must share identical source metadata")
	}

	expected, err := BuildRuleCatalog(spec, catalog, templates)
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

func shouldOmitRequiredRule(spec NormalizedSpec, bodySchema *NormalizedSchema, rule SemanticRule) bool {
	if bodySchema == nil || len(rule.Require) == 0 {
		return true
	}
	for _, requirement := range rule.Require {
		if requirement.MinCount > 0 {
			return false
		}
		if !fieldRequiredBySchema(spec, bodySchema, requirement.Field) {
			return false
		}
	}
	return true
}

func fieldRequiredBySchema(spec NormalizedSpec, root *NormalizedSchema, fieldPath string) bool {
	current := dereferenceSchema(spec, root)
	if current == nil {
		return false
	}
	segments := strings.Split(strings.TrimSpace(fieldPath), ".")
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		current = dereferenceSchema(spec, current)
		if current == nil || !slices.Contains(current.Required, segment) {
			return false
		}
		next, ok := current.Properties[segment]
		if !ok {
			return false
		}
		current = next
	}
	return true
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

func NewRequiredRule(field, target string, minCount ...int) SemanticRule {
	requirement := FieldRule{Field: field, Target: target}
	if len(minCount) > 0 {
		requirement.MinCount = minCount[0]
	}
	return SemanticRule{
		Kind:    "required",
		Require: []FieldRule{requirement},
	}
}

func NewConditionalRequireRule(field string, equals any, requirement FieldRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, Equals: equals},
		Require: []FieldRule{requirement},
	}
}

func NewConditionalInRequireRule(field string, values []any, requirement FieldRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, In: append([]any(nil), values...)},
		Require: []FieldRule{requirement},
	}
}

func NewConditionalForbidRule(field string, equals any, forbidden string) SemanticRule {
	return SemanticRule{
		Kind:   "conditional",
		When:   &RuleCondition{Field: field, Equals: equals},
		Forbid: []string{forbidden},
	}
}

func NewForbidRule(field string) SemanticRule {
	return SemanticRule{
		Kind:   "forbidden",
		Forbid: []string{field},
	}
}

func NewMinimumRule(minimum MinimumRule) SemanticRule {
	return SemanticRule{
		Kind:    "minimum",
		Minimum: []MinimumRule{minimum},
	}
}

func NewOneOfRule(fields ...string) SemanticRule {
	requireAny := make([]FieldRule, 0, len(fields))
	for _, field := range fields {
		requireAny = append(requireAny, FieldRule{Field: field})
	}
	return SemanticRule{
		Kind:       "one_of",
		RequireAny: requireAny,
	}
}

func NewConditionalMinimumRule(field string, equals any, minimum MinimumRule) SemanticRule {
	return SemanticRule{
		Kind:    "conditional",
		When:    &RuleCondition{Field: field, Equals: equals},
		Minimum: []MinimumRule{minimum},
	}
}
