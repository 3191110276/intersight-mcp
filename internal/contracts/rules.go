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
	RequireEach []FieldRule    `json:"requireEach,omitempty"`
	RequireAny  []FieldRule    `json:"requireAny,omitempty"`
	Forbid      []string       `json:"forbid,omitempty"`
	Minimum     []MinimumRule  `json:"minimum,omitempty"`
	Maximum     []LengthRule   `json:"maximum,omitempty"`
	Pattern     []PatternRule  `json:"pattern,omitempty"`
	Future      []TimeRule     `json:"future,omitempty"`
	Contains    []ContainsRule `json:"contains,omitempty"`
	Custom      []CustomRule   `json:"custom,omitempty"`
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

type LengthRule struct {
	Field string `json:"field"`
	Value int    `json:"value"`
}

type PatternRule struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type TimeRule struct {
	Field string `json:"field"`
}

type ContainsRule struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type CustomRule struct {
	Field     string `json:"field"`
	Validator string `json:"validator"`
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
		methods := resolveRuleTemplateMethods(catalog, entry)
		if len(methods) == 0 {
			continue
		}
		for _, method := range methods {
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
	}

	return rules, nil
}

func resolveRuleTemplateMethods(catalog SDKCatalog, entry RuleTemplate) []SDKMethod {
	if method, ok := catalog.Methods[entry.SDKMethod]; ok {
		return ruleTemplateAliasMethods(catalog, method)
	}

	verb := sdkMethodVerb(entry.SDKMethod)
	if verb == "" || strings.TrimSpace(entry.Resource) == "" {
		return nil
	}

	var matches []SDKMethod
	var operationIDs = map[string]struct{}{}
	for _, method := range catalog.Methods {
		if method.Resource != entry.Resource || sdkMethodVerb(method.SDKMethod) != verb {
			continue
		}
		matches = append(matches, method)
		operationIDs[method.Descriptor.OperationID] = struct{}{}
	}
	if len(matches) == 0 {
		return nil
	}
	if len(operationIDs) != 1 {
		if len(matches) != 1 {
			return nil
		}
		return []SDKMethod{matches[0]}
	}
	return normalizeRuleTemplateMatches(matches)
}

func ruleTemplateAliasMethods(catalog SDKCatalog, canonical SDKMethod) []SDKMethod {
	var matches []SDKMethod
	for _, method := range catalog.Methods {
		if method.Descriptor.OperationID != canonical.Descriptor.OperationID {
			continue
		}
		if method.Resource != canonical.Resource {
			continue
		}
		if sdkMethodVerb(method.SDKMethod) != sdkMethodVerb(canonical.SDKMethod) {
			continue
		}
		matches = append(matches, method)
	}
	if len(matches) == 0 {
		return []SDKMethod{canonical}
	}
	return normalizeRuleTemplateMatches(matches)
}

func normalizeRuleTemplateMatches(matches []SDKMethod) []SDKMethod {
	slices.SortFunc(matches, func(a, b SDKMethod) int {
		return strings.Compare(a.SDKMethod, b.SDKMethod)
	})
	return matches
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
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown required field %q", sdkMethod, requirement.Field)
			}
			if ok && requirement.Target != "" {
				if _, ok := spec.Schemas[requirement.Target]; !ok {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q points at unknown relationship target %q", sdkMethod, requirement.Target)
				}
				if err := validateRelationshipTarget(requirement.Target, schema); err != nil {
					return fmt.Errorf("embedded artifact validation failed: rules entry %q field %q %w", sdkMethod, requirement.Field, err)
				}
			}
		}
		for _, requirement := range rule.RequireEach {
			if _, ok := schemaAtFieldPath(spec, bodySchema, requirement.Field); !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown per-item required field %q", sdkMethod, requirement.Field)
			}
		}
		for _, requirement := range rule.RequireAny {
			schema, ok := schemaAtFieldPath(spec, bodySchema, requirement.Field)
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown one-of field %q", sdkMethod, requirement.Field)
			}
			if ok && requirement.Target != "" {
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
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown minimum field %q", sdkMethod, minimum.Field)
			}
			if !ok {
				continue
			}
			switch schema.Type {
			case "integer", "number":
			default:
				return fmt.Errorf("embedded artifact validation failed: rules entry %q minimum field %q must resolve to a numeric schema", sdkMethod, minimum.Field)
			}
		}
		for _, maximum := range rule.Maximum {
			schema, ok := schemaAtFieldPath(spec, bodySchema, maximum.Field)
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown maximum field %q", sdkMethod, maximum.Field)
			}
			if ok && schema.Type != "string" {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q maximum field %q must resolve to a string schema", sdkMethod, maximum.Field)
			}
		}
		for _, pattern := range rule.Pattern {
			schema, ok := schemaAtFieldPath(spec, bodySchema, pattern.Field)
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown pattern field %q", sdkMethod, pattern.Field)
			}
			if ok && schema.Type != "string" {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q pattern field %q must resolve to a string schema", sdkMethod, pattern.Field)
			}
		}
		for _, future := range rule.Future {
			schema, ok := schemaAtFieldPath(spec, bodySchema, future.Field)
			if !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown future field %q", sdkMethod, future.Field)
			}
			if ok && schema.Type != "string" {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q future field %q must resolve to a string schema", sdkMethod, future.Field)
			}
		}
		for _, contains := range rule.Contains {
			if _, ok := schemaAtFieldPath(spec, bodySchema, contains.Field); !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown contains field %q", sdkMethod, contains.Field)
			}
		}
		for _, custom := range rule.Custom {
			if _, ok := schemaAtFieldPath(spec, bodySchema, custom.Field); !ok && !allowUnknownRuleField(spec, bodySchema) {
				return fmt.Errorf("embedded artifact validation failed: rules entry %q references unknown custom field %q", sdkMethod, custom.Field)
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
	fieldPath = strings.TrimSpace(fieldPath)
	if fieldPath == "." || fieldPath == "$" {
		return dereferenceSchema(spec, root), root != nil
	}

	current := root
	for _, segment := range strings.Split(fieldPath, ".") {
		if segment == "" {
			return nil, false
		}
		arrayItem := strings.HasSuffix(segment, "[]")
		if arrayItem {
			segment = strings.TrimSuffix(segment, "[]")
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
		if arrayItem {
			current = dereferenceSchema(spec, current)
			if current == nil || current.Items == nil {
				return nil, false
			}
			current = current.Items
		}
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
		out[i].RequireEach = append([]FieldRule(nil), out[i].RequireEach...)
		slices.SortFunc(out[i].RequireEach, func(a, b FieldRule) int {
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
		out[i].Maximum = append([]LengthRule(nil), out[i].Maximum...)
		slices.SortFunc(out[i].Maximum, func(a, b LengthRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].Pattern = append([]PatternRule(nil), out[i].Pattern...)
		slices.SortFunc(out[i].Pattern, func(a, b PatternRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].Future = append([]TimeRule(nil), out[i].Future...)
		slices.SortFunc(out[i].Future, func(a, b TimeRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].Contains = append([]ContainsRule(nil), out[i].Contains...)
		slices.SortFunc(out[i].Contains, func(a, b ContainsRule) int {
			return strings.Compare(a.Field, b.Field)
		})
		out[i].Custom = append([]CustomRule(nil), out[i].Custom...)
		slices.SortFunc(out[i].Custom, func(a, b CustomRule) int {
			if cmp := strings.Compare(a.Field, b.Field); cmp != 0 {
				return cmp
			}
			return strings.Compare(a.Validator, b.Validator)
		})
		if out[i].When != nil && len(out[i].When.In) > 0 {
			out[i].When.In = append([]any(nil), out[i].When.In...)
		}
	}
	return out
}

func allowUnknownRuleField(spec NormalizedSpec, bodySchema *NormalizedSchema) bool {
	bodySchema = dereferenceSchema(spec, bodySchema)
	return bodySchema != nil && bodySchema.Type == "object" && len(bodySchema.Properties) == 0
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

func NewConditionalCustomRule(field string, equals any, custom CustomRule) SemanticRule {
	return SemanticRule{
		Kind:   "conditional",
		When:   &RuleCondition{Field: field, Equals: equals},
		Custom: []CustomRule{custom},
	}
}

func NewConditionalInCustomRule(field string, values []any, custom CustomRule) SemanticRule {
	return SemanticRule{
		Kind:   "conditional",
		When:   &RuleCondition{Field: field, In: append([]any(nil), values...)},
		Custom: []CustomRule{custom},
	}
}

func NewEachRequiredRule(field string) SemanticRule {
	return SemanticRule{
		Kind:        "required_each",
		RequireEach: []FieldRule{{Field: field}},
	}
}

func NewMaximumRule(maximum LengthRule) SemanticRule {
	return SemanticRule{
		Kind:    "maximum",
		Maximum: []LengthRule{maximum},
	}
}

func NewPatternRule(pattern PatternRule) SemanticRule {
	return SemanticRule{
		Kind:    "pattern",
		Pattern: []PatternRule{pattern},
	}
}

func NewFutureRule(field string) SemanticRule {
	return SemanticRule{
		Kind:   "future",
		Future: []TimeRule{{Field: field}},
	}
}

func NewContainsRule(contains ContainsRule) SemanticRule {
	return SemanticRule{
		Kind:     "contains",
		Contains: []ContainsRule{contains},
	}
}

func NewCustomRule(custom CustomRule) SemanticRule {
	return SemanticRule{
		Kind:   "custom",
		Custom: []CustomRule{custom},
	}
}
