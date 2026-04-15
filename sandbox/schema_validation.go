package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

type dryRunSpecIndex struct {
	Paths   map[string]map[string]dryRunOperation `json:"paths"`
	Schemas map[string]dryRunSchema               `json:"schemas"`
	ext     Extensions
}

type dryRunOperation struct {
	RequestBody *dryRunRequestBody `json:"requestBody,omitempty"`
}

type dryRunRequestBody struct {
	Required bool                          `json:"required"`
	Content  map[string]dryRunMediaContent `json:"content"`
}

type dryRunMediaContent struct {
	Schema *dryRunSchema `json:"schema,omitempty"`
}

type dryRunSchema struct {
	Type                   string                   `json:"type,omitempty"`
	Format                 string                   `json:"format,omitempty"`
	Pattern                string                   `json:"pattern,omitempty"`
	Properties             map[string]*dryRunSchema `json:"properties,omitempty"`
	Required               []string                 `json:"required,omitempty"`
	Items                  *dryRunSchema            `json:"items,omitempty"`
	Nullable               bool                     `json:"nullable,omitempty"`
	AdditionalProperties   json.RawMessage          `json:"additionalProperties,omitempty"`
	OneOf                  []*dryRunSchema          `json:"oneOf,omitempty"`
	AnyOf                  []*dryRunSchema          `json:"anyOf,omitempty"`
	Circular               string                   `json:"$circular,omitempty"`
	ExpandTarget           string                   `json:"$expandTarget,omitempty"`
	Relationship           bool                     `json:"x-relationship,omitempty"`
	RelationshipTarget     string                   `json:"x-relationshipTarget,omitempty"`
	RelationshipWriteForms []string                 `json:"x-writeForms,omitempty"`
	Enum                   []any                    `json:"enum,omitempty"`
}

type dryRunValidationError struct {
	Path      string `json:"path"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Rule      string `json:"rule,omitempty"`
	Message   string `json:"message"`
	Condition string `json:"condition,omitempty"`
	Expected  any    `json:"expected,omitempty"`
	Actual    any    `json:"actual,omitempty"`
}

type schemaValidationState struct {
	visiting map[string]int
}

func loadDryRunSpecIndex(specJSON []byte, ext Extensions) (*dryRunSpecIndex, error) {
	if len(specJSON) == 0 {
		return nil, nil
	}
	if !json.Valid(specJSON) {
		return nil, contracts.ValidationError{Message: "embedded spec is not valid JSON"}
	}

	var spec dryRunSpecIndex
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, contracts.ValidationError{Message: "decode embedded spec", Err: err}
	}
	spec.ext = normalizeExtensions(ext)
	return &spec, nil
}

func (s *dryRunSpecIndex) requestSchema(method, requestPath string) *dryRunSchema {
	if s == nil {
		return nil
	}
	op := s.operation(method, requestPath)
	if op == nil || op.RequestBody == nil {
		return nil
	}
	content := op.RequestBody.Content["application/json"]
	return content.Schema
}

func (s *dryRunSpecIndex) operation(method, requestPath string) *dryRunOperation {
	if s == nil {
		return nil
	}

	cleanPath := normalizeRequestPath(requestPath)
	methods := s.Paths[cleanPath]
	if op, ok := methods[strings.ToLower(strings.TrimSpace(method))]; ok {
		return &op
	}

	for candidatePath, candidateMethods := range s.Paths {
		if !specPathMatchesRequest(candidatePath, cleanPath) {
			continue
		}
		if op, ok := candidateMethods[strings.ToLower(strings.TrimSpace(method))]; ok {
			return &op
		}
	}
	return nil
}

func normalizeRequestPath(requestPath string) string {
	clean := path.Clean("/" + strings.TrimSpace(requestPath))
	if clean == "." {
		return "/"
	}
	return clean
}

func specPathMatchesRequest(specPath, requestPath string) bool {
	specParts := strings.Split(strings.Trim(specPath, "/"), "/")
	reqParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(specParts) != len(reqParts) {
		return false
	}
	for i := range specParts {
		part := specParts[i]
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		if part != reqParts[i] {
			return false
		}
	}
	return true
}

func validateRequestBodyAgainstSchema(spec *dryRunSpecIndex, schema *dryRunSchema, body any) []dryRunValidationError {
	if schema == nil {
		return nil
	}
	state := &schemaValidationState{visiting: map[string]int{}}
	return validateValueAgainstSchema(spec, schema, body, "$", state)
}

func normalizeValueForSchema(spec *dryRunSpecIndex, schema *dryRunSchema, value any, state *schemaValidationState) any {
	if schema == nil {
		return value
	}

	if schema.Circular != "" {
		if state.visiting[schema.Circular] > 0 {
			return value
		}
		target, ok := spec.Schemas[schema.Circular]
		if !ok {
			return value
		}
		state.visiting[schema.Circular]++
		defer func() {
			state.visiting[schema.Circular]--
		}()
		return normalizeValueForSchema(spec, &target, value, state)
	}

	if schema.Relationship && spec.ext.RelationshipBehavior != nil {
		return normalizeRelationshipValue(spec.ext.RelationshipBehavior, schema, value)
	}

	switch typed := value.(type) {
	case map[string]any:
		out := typed
		if spec.ext.AutofillDiscriminators {
			out = normalizeObjectDiscriminators(schema, typed)
		}
		for key, child := range out {
			childSchema := schema.Properties[key]
			out[key] = normalizeValueForSchema(spec, childSchema, child, state)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = normalizeValueForSchema(spec, schema.Items, child, state)
		}
		return out
	default:
		return value
	}
}

func normalizeObjectDiscriminators(schema *dryRunSchema, value map[string]any) map[string]any {
	out := make(map[string]any, len(value)+2)
	for key, child := range value {
		out[key] = child
	}

	if strings.TrimSpace(stringValue(out["ClassId"])) == "" {
		if classID, ok := singletonStringEnumProperty(schema, "ClassId"); ok {
			out["ClassId"] = classID
		}
	}
	if strings.TrimSpace(stringValue(out["ObjectType"])) == "" {
		if objectType, ok := singletonStringEnumProperty(schema, "ObjectType"); ok {
			out["ObjectType"] = objectType
		}
	}

	return out
}

func singletonStringEnumProperty(schema *dryRunSchema, name string) (string, bool) {
	if schema == nil || schema.Properties == nil {
		return "", false
	}
	property := schema.Properties[name]
	if property == nil || len(property.Enum) != 1 {
		return "", false
	}
	value, ok := property.Enum[0].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

func normalizeRelationshipValue(behavior *RelationshipBehavior, schema *dryRunSchema, value any) any {
	obj, ok := value.(map[string]any)
	if !ok {
		return value
	}

	out := make(map[string]any, len(obj)+2)
	for key, child := range obj {
		out[key] = child
	}
	if strings.TrimSpace(stringValue(out["Selector"])) != "" {
		return out
	}
	if strings.TrimSpace(stringValue(out[behavior.MoidField])) == "" {
		return out
	}
	if behavior.AutofillTargetObjectType && strings.TrimSpace(stringValue(out[behavior.ObjectTypeField])) == "" && schema.RelationshipTarget != "" {
		out[behavior.ObjectTypeField] = schema.RelationshipTarget
	}
	if strings.TrimSpace(stringValue(out[behavior.ClassIDField])) == "" && strings.TrimSpace(behavior.DefaultClassID) != "" {
		out[behavior.ClassIDField] = behavior.DefaultClassID
	}
	return out
}

func validateValueAgainstSchema(spec *dryRunSpecIndex, schema *dryRunSchema, value any, fieldPath string, state *schemaValidationState) []dryRunValidationError {
	if schema == nil {
		return nil
	}

	if schema.Circular != "" {
		if state.visiting[schema.Circular] > 0 {
			return nil
		}
		target, ok := spec.Schemas[schema.Circular]
		if !ok {
			return []dryRunValidationError{newOpenAPIIssue(fieldPath, "schema_reference", "schema-reference", fmt.Sprintf("Schema reference %q could not be resolved.", schema.Circular))}
		}
		state.visiting[schema.Circular]++
		defer func() {
			state.visiting[schema.Circular]--
		}()
		return validateValueAgainstSchema(spec, &target, value, fieldPath, state)
	}

	if value == nil {
		if schema.Nullable {
			return nil
		}
		return []dryRunValidationError{newOpenAPIIssue(fieldPath, "nullable", "nullable", "Field does not allow null.")}
	}

	if schema.Relationship && spec.ext.RelationshipBehavior != nil {
		return validateRelationshipValue(spec.ext.RelationshipBehavior, schema, value, fieldPath)
	}

	if len(schema.OneOf) > 0 {
		matches := 0
		for _, option := range schema.OneOf {
			if len(validateValueAgainstSchema(spec, option, value, fieldPath, &schemaValidationState{visiting: copyVisiting(state.visiting)})) == 0 {
				matches++
			}
		}
		if matches != 1 {
			return []dryRunValidationError{newOpenAPIIssue(fieldPath, "one_of", "oneOf", fmt.Sprintf("Value must match exactly one schema option, matched %d.", matches))}
		}
	}

	if len(schema.AnyOf) > 0 {
		for _, option := range schema.AnyOf {
			if len(validateValueAgainstSchema(spec, option, value, fieldPath, &schemaValidationState{visiting: copyVisiting(state.visiting)})) == 0 {
				goto anyOfMatched
			}
		}
		return []dryRunValidationError{newOpenAPIIssue(fieldPath, "any_of", "anyOf", "Value must match at least one schema option.")}
	}

anyOfMatched:
	if len(schema.Enum) > 0 && !valueMatchesEnum(value, schema.Enum) {
		return []dryRunValidationError{{
			Path:     fieldPath,
			Type:     "enum",
			Source:   validationSourceOpenAPI,
			Rule:     "enum",
			Message:  fmt.Sprintf("Value %s is not one of the allowed enum values.", formatValue(value)),
			Expected: schema.Enum,
			Actual:   value,
		}}
	}

	switch inferredSchemaType(schema) {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "object", value)}
		}
		return validateObject(spec, schema, obj, fieldPath, state)
	case "array":
		items, ok := value.([]any)
		if !ok {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "array", value)}
		}
		return validateArray(spec, schema, items, fieldPath, state)
	case "string":
		stringValue, ok := value.(string)
		if !ok {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "string", value)}
		}
		if pattern := strings.TrimSpace(schema.Pattern); pattern != "" {
			matched, err := regexp.MatchString(pattern, stringValue)
			if err != nil {
				return []dryRunValidationError{newOpenAPIIssue(fieldPath, "pattern_invalid", "pattern", fmt.Sprintf("Schema pattern %q could not be compiled.", pattern))}
			}
			if !matched {
				return []dryRunValidationError{{
					Path:      fieldPath,
					Type:      "pattern",
					Source:    validationSourceOpenAPI,
					Rule:      "pattern",
					Message:   fmt.Sprintf("String value does not match required pattern %q.", pattern),
					Condition: pattern,
					Actual:    stringValue,
				}}
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "boolean", value)}
		}
	case "integer":
		if !isInteger(value) {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "integer", value)}
		}
	case "number":
		if !isNumber(value) {
			return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "number", value)}
		}
	}

	return nil
}

func validateRelationshipValue(behavior *RelationshipBehavior, schema *dryRunSchema, value any, fieldPath string) []dryRunValidationError {
	obj, ok := value.(map[string]any)
	if !ok {
		return []dryRunValidationError{newTypeMismatchIssue(fieldPath, "object", value)}
	}

	if behavior.RejectSelector {
		if selector := strings.TrimSpace(stringValue(obj["Selector"])); selector != "" {
			return []dryRunValidationError{newOpenAPIIssue(fieldPath, "relationship", behavior.RelationshipRuleName, behavior.SelectorMessage)}
		}
	}

	classID := strings.TrimSpace(stringValue(obj[behavior.ClassIDField]))
	objectType := strings.TrimSpace(stringValue(obj[behavior.ObjectTypeField]))
	moid := strings.TrimSpace(stringValue(obj[behavior.MoidField]))
	allowMoidRef := relationshipWriteFormAllowed(schema, behavior.AllowMoidRefWriteForm)
	allowTypedMoRef := relationshipWriteFormAllowed(schema, behavior.AllowTypedMoRefWriteForm)

	if moid == "" {
		return []dryRunValidationError{newOpenAPIIssue(joinFieldPath(fieldPath, behavior.MoidField), "relationship", behavior.RelationshipRuleName, behavior.MissingMoidMessage)}
	}

	if classID == "" && objectType == "" && allowMoidRef {
		return nil
	}

	if !allowTypedMoRef {
		return []dryRunValidationError{newOpenAPIIssue(fieldPath, "relationship", behavior.RelationshipRuleName, behavior.InvalidPayloadShapeMessage)}
	}

	if behavior.RequiredClassID != "" && classID != behavior.RequiredClassID {
		return []dryRunValidationError{{
			Path:     joinFieldPath(fieldPath, behavior.ClassIDField),
			Type:     "relationship",
			Source:   validationSourceOpenAPI,
			Rule:     behavior.RelationshipRuleName,
			Message:  fmt.Sprintf("Relationship %s must be %q.", behavior.ClassIDField, behavior.RequiredClassID),
			Expected: behavior.RequiredClassID,
			Actual:   classID,
		}}
	}
	if schema.RelationshipTarget != "" && objectType != schema.RelationshipTarget {
		return []dryRunValidationError{{
			Path:     joinFieldPath(fieldPath, behavior.ObjectTypeField),
			Type:     "relationship",
			Source:   validationSourceOpenAPI,
			Rule:     behavior.RelationshipRuleName,
			Message:  fmt.Sprintf("Relationship %s must be %q.", behavior.ObjectTypeField, schema.RelationshipTarget),
			Expected: schema.RelationshipTarget,
			Actual:   objectType,
		}}
	}
	return nil
}

func relationshipWriteFormAllowed(schema *dryRunSchema, form string) bool {
	if schema == nil {
		return false
	}
	if len(schema.RelationshipWriteForms) == 0 {
		return form == "typedMoRef"
	}
	for _, candidate := range schema.RelationshipWriteForms {
		if strings.EqualFold(strings.TrimSpace(candidate), form) {
			return true
		}
	}
	return false
}

func validateObject(spec *dryRunSpecIndex, schema *dryRunSchema, obj map[string]any, fieldPath string, state *schemaValidationState) []dryRunValidationError {
	var errs []dryRunValidationError
	for _, required := range schema.Required {
		if _, ok := obj[required]; !ok {
			errs = append(errs, newOpenAPIIssue(joinFieldPath(fieldPath, required), "required", "required", "Required field is missing."))
		}
	}

	for key, value := range obj {
		childPath := joinFieldPath(fieldPath, key)
		if childSchema, ok := schema.Properties[key]; ok && childSchema != nil {
			errs = append(errs, validateValueAgainstSchema(spec, childSchema, value, childPath, state)...)
			continue
		}

		allowed, additionalSchema, err := decodeAdditionalProperties(schema.AdditionalProperties)
		if err != nil {
			errs = append(errs, newOpenAPIIssue(childPath, "unknown_field", "additionalProperties", err.Error()))
			continue
		}
		if !allowed {
			errs = append(errs, newOpenAPIIssue(childPath, "unknown_field", "additionalProperties", "Additional property is not allowed."))
			continue
		}
		if additionalSchema != nil {
			errs = append(errs, validateValueAgainstSchema(spec, additionalSchema, value, childPath, state)...)
		}
	}
	return errs
}

func newOpenAPIIssue(path, issueType, rule, message string) dryRunValidationError {
	return dryRunValidationError{
		Path:    path,
		Type:    issueType,
		Source:  validationSourceOpenAPI,
		Rule:    rule,
		Message: message,
	}
}

func newTypeMismatchIssue(path, expected string, actual any) dryRunValidationError {
	return dryRunValidationError{
		Path:     path,
		Type:     "type_mismatch",
		Source:   validationSourceOpenAPI,
		Rule:     "type",
		Message:  fmt.Sprintf("Expected %s, got %T.", expected, actual),
		Expected: expected,
		Actual:   fmt.Sprintf("%T", actual),
	}
}

func validateArray(spec *dryRunSpecIndex, schema *dryRunSchema, items []any, fieldPath string, state *schemaValidationState) []dryRunValidationError {
	if schema.Items == nil {
		return nil
	}
	var errs []dryRunValidationError
	for i, item := range items {
		errs = append(errs, validateValueAgainstSchema(spec, schema.Items, item, fmt.Sprintf("%s[%d]", fieldPath, i), state)...)
	}
	return errs
}

func decodeAdditionalProperties(raw json.RawMessage) (bool, *dryRunSchema, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return true, nil, nil
	}

	var allowed bool
	if err := json.Unmarshal(raw, &allowed); err == nil {
		return allowed, nil, nil
	}

	var schema dryRunSchema
	if err := json.Unmarshal(raw, &schema); err == nil {
		return true, &schema, nil
	}

	return false, nil, fmt.Errorf("could not decode additionalProperties schema")
}

func inferredSchemaType(schema *dryRunSchema) string {
	if schema == nil {
		return ""
	}
	if schema.Type != "" {
		return schema.Type
	}
	switch {
	case len(schema.Properties) > 0 || len(schema.Required) > 0:
		return "object"
	case schema.Items != nil:
		return "array"
	default:
		return ""
	}
}

func isInteger(value any) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return float32(int64(v)) == v
	case float64:
		return float64(int64(v)) == v
	default:
		return false
	}
}

func isNumber(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

func valueMatchesEnum(value any, allowed []any) bool {
	for _, candidate := range allowed {
		if reflect.DeepEqual(value, candidate) {
			return true
		}
		if numbersEqual(value, candidate) {
			return true
		}
	}
	return false
}

func numbersEqual(left, right any) bool {
	if !isNumber(left) || !isNumber(right) {
		return false
	}
	lf, lok := asFloat64(left)
	rf, rok := asFloat64(right)
	return lok && rok && lf == rf
}

func formatValue(value any) string {
	rendered, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(rendered)
}

func joinFieldPath(base, child string) string {
	if base == "" || base == "$" {
		return "$." + child
	}
	return base + "." + child
}

func copyVisiting(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
