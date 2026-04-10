package sandbox

import (
	"fmt"
	"math"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func (r *sdkRuntime) validateSemanticRules(sdkMethod string, body any) []dryRunValidationError {
	methodRules, ok := r.rules.Methods[sdkMethod]
	if !ok || len(methodRules.Rules) == 0 {
		return nil
	}
	bodyMap, ok := body.(map[string]any)
	if !ok {
		return nil
	}

	var errs []dryRunValidationError
	for _, rule := range methodRules.Rules {
		if !matchesCondition(bodyMap, rule.When) {
			continue
		}
		condition := describeCondition(rule.When)
		for _, requirement := range rule.Require {
			if present, count := fieldPresence(bodyMap, requirement.Field); !present || (requirement.MinCount > 0 && count < requirement.MinCount) {
				errs = append(errs, dryRunValidationError{
					Path:      requirement.Field,
					Type:      semanticRequirementType(requirement),
					Source:    validationSourceRules,
					Rule:      "require",
					Message:   missingFieldMessage(requirement),
					Condition: condition,
				})
			}
		}
		if len(rule.RequireAny) > 0 {
			matched := false
			for _, requirement := range rule.RequireAny {
				if present, count := fieldPresence(bodyMap, requirement.Field); present && (requirement.MinCount == 0 || count >= requirement.MinCount) {
					matched = true
					break
				}
			}
			if !matched {
				fields := make([]string, 0, len(rule.RequireAny))
				for _, requirement := range rule.RequireAny {
					fields = append(fields, requirement.Field)
				}
				errs = append(errs, dryRunValidationError{
					Path:      strings.Join(fields, "|"),
					Type:      "one_of",
					Source:    validationSourceRules,
					Rule:      "requireAny",
					Message:   fmt.Sprintf("At least one of [%s] must be provided.", strings.Join(fields, ", ")),
					Condition: condition,
				})
			}
		}
		for _, field := range rule.Forbid {
			if present, _ := fieldPresence(bodyMap, field); present {
				errs = append(errs, dryRunValidationError{
					Path:      field,
					Type:      "forbidden",
					Source:    validationSourceRules,
					Rule:      "forbid",
					Message:   fmt.Sprintf("Field %q must not be provided when %s.", field, condition),
					Condition: condition,
				})
			}
		}
		for _, minimum := range rule.Minimum {
			value, ok := fieldValue(bodyMap, minimum.Field)
			if !ok {
				errs = append(errs, dryRunValidationError{
					Path:      minimum.Field,
					Type:      "minimum",
					Source:    validationSourceRules,
					Rule:      "minimum",
					Message:   fmt.Sprintf("Field %q must be provided when %s.", minimum.Field, condition),
					Condition: condition,
					Expected:  minimum.Value,
				})
				continue
			}
			number, ok := asFloat64(value)
			if !ok || number < minimum.Value {
				errs = append(errs, dryRunValidationError{
					Path:      minimum.Field,
					Type:      "minimum",
					Source:    validationSourceRules,
					Rule:      "minimum",
					Message:   fmt.Sprintf("Field %q must be at least %s when %s.", minimum.Field, trimFloat(minimum.Value), condition),
					Condition: condition,
					Expected:  minimum.Value,
					Actual:    value,
				})
			}
		}
	}
	return errs
}

func semanticRequirementType(requirement contracts.FieldRule) string {
	if requirement.MinCount > 0 {
		return "min_items"
	}
	return "required"
}

func matchesCondition(body map[string]any, condition *contracts.RuleCondition) bool {
	if condition == nil {
		return true
	}
	value, ok := fieldValue(body, condition.Field)
	if !ok {
		return false
	}
	if len(condition.In) > 0 {
		for _, candidate := range condition.In {
			if valuesEqual(value, candidate) {
				return true
			}
		}
		return false
	}
	return valuesEqual(value, condition.Equals)
}

func fieldPresence(body map[string]any, fieldPath string) (bool, int) {
	value, ok := fieldValue(body, fieldPath)
	if !ok || value == nil {
		return false, 0
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != "", 1
	case []any:
		return len(typed) > 0, len(typed)
	default:
		return true, 1
	}
}

func fieldValue(body map[string]any, fieldPath string) (any, bool) {
	var current any = body
	for _, segment := range strings.Split(strings.TrimSpace(fieldPath), ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func missingFieldMessage(requirement contracts.FieldRule) string {
	if requirement.MinCount > 0 {
		return fmt.Sprintf("Field %q must include at least %d entrie(s).", requirement.Field, requirement.MinCount)
	}
	if requirement.Target != "" {
		return fmt.Sprintf("Field %q must reference %q.", requirement.Field, requirement.Target)
	}
	return fmt.Sprintf("Field %q is required.", requirement.Field)
}

func describeCondition(condition *contracts.RuleCondition) string {
	if condition == nil {
		return "the current request is evaluated"
	}
	if len(condition.In) > 0 {
		parts := make([]string, 0, len(condition.In))
		for _, candidate := range condition.In {
			parts = append(parts, fmt.Sprintf("%v", candidate))
		}
		return fmt.Sprintf("%q is one of [%s]", condition.Field, strings.Join(parts, ", "))
	}
	return fmt.Sprintf("%q equals %v", condition.Field, condition.Equals)
}

func valuesEqual(left, right any) bool {
	switch l := left.(type) {
	case string:
		r, ok := right.(string)
		return ok && l == r
	case bool:
		r, ok := right.(bool)
		return ok && l == r
	default:
		lf, lok := asFloat64(left)
		rf, rok := asFloat64(right)
		return lok && rok && lf == rf
	}
}

func asFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	default:
		return 0, false
	}
}

func trimFloat(value float64) string {
	if value == math.Trunc(value) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%g", value)
}
