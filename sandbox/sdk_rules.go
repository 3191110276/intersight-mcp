package sandbox

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

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
		for _, requirement := range rule.RequireEach {
			if missing := missingEachField(bodyMap, requirement.Field); len(missing) > 0 {
				errs = append(errs, dryRunValidationError{
					Path:      requirement.Field,
					Type:      "required_each",
					Source:    validationSourceRules,
					Rule:      "requireEach",
					Message:   fmt.Sprintf("Field %q must be present for every matching entry.", requirement.Field),
					Condition: condition,
					Actual:    missing,
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
		for _, maximum := range rule.Maximum {
			values, ok := fieldValues(bodyMap, maximum.Field)
			if !ok {
				continue
			}
			for _, value := range values {
				text, ok := value.(string)
				if !ok || len(text) <= maximum.Value {
					continue
				}
				errs = append(errs, dryRunValidationError{
					Path:      maximum.Field,
					Type:      "maximum",
					Source:    validationSourceRules,
					Rule:      "maximum",
					Message:   fmt.Sprintf("Field %q must not exceed %d characters when %s.", maximum.Field, maximum.Value, condition),
					Condition: condition,
					Expected:  maximum.Value,
					Actual:    len(text),
				})
				break
			}
		}
		for _, patternRule := range rule.Pattern {
			values, ok := fieldValues(bodyMap, patternRule.Field)
			if !ok {
				continue
			}
			re, err := regexp.Compile(patternRule.Value)
			if err != nil {
				errs = append(errs, dryRunValidationError{
					Path:      patternRule.Field,
					Type:      "pattern_invalid",
					Source:    validationSourceRules,
					Rule:      "pattern",
					Message:   fmt.Sprintf("Rule pattern %q could not be compiled.", patternRule.Value),
					Condition: condition,
					Expected:  patternRule.Value,
				})
				continue
			}
			for _, value := range values {
				text, ok := value.(string)
				if ok && re.MatchString(text) {
					continue
				}
				errs = append(errs, dryRunValidationError{
					Path:      patternRule.Field,
					Type:      "pattern",
					Source:    validationSourceRules,
					Rule:      "pattern",
					Message:   fmt.Sprintf("Field %q must match %q when %s.", patternRule.Field, patternRule.Value, condition),
					Condition: condition,
					Expected:  patternRule.Value,
					Actual:    value,
				})
				break
			}
		}
		for _, futureRule := range rule.Future {
			values, ok := fieldValues(bodyMap, futureRule.Field)
			if !ok {
				continue
			}
			now := time.Now().UTC()
			for _, value := range values {
				text, ok := value.(string)
				if !ok {
					errs = append(errs, dryRunValidationError{
						Path:      futureRule.Field,
						Type:      "future",
						Source:    validationSourceRules,
						Rule:      "future",
						Message:   fmt.Sprintf("Field %q must be a future timestamp when %s.", futureRule.Field, condition),
						Condition: condition,
						Actual:    value,
					})
					break
				}
				parsed, err := time.Parse(time.RFC3339, text)
				if err != nil || !parsed.After(now) {
					errs = append(errs, dryRunValidationError{
						Path:      futureRule.Field,
						Type:      "future",
						Source:    validationSourceRules,
						Rule:      "future",
						Message:   fmt.Sprintf("Field %q must be a future timestamp when %s.", futureRule.Field, condition),
						Condition: condition,
						Actual:    value,
					})
					break
				}
			}
		}
		for _, containsRule := range rule.Contains {
			values, ok := fieldValues(bodyMap, containsRule.Field)
			if !ok {
				errs = append(errs, dryRunValidationError{
					Path:      containsRule.Field,
					Type:      "contains",
					Source:    validationSourceRules,
					Rule:      "contains",
					Message:   fmt.Sprintf("Field %q must contain %v when %s.", containsRule.Field, containsRule.Value, condition),
					Condition: condition,
					Expected:  containsRule.Value,
				})
				continue
			}
			found := false
			for _, value := range values {
				if valuesEqual(value, containsRule.Value) {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, dryRunValidationError{
					Path:      containsRule.Field,
					Type:      "contains",
					Source:    validationSourceRules,
					Rule:      "contains",
					Message:   fmt.Sprintf("Field %q must contain %v when %s.", containsRule.Field, containsRule.Value, condition),
					Condition: condition,
					Expected:  containsRule.Value,
					Actual:    values,
				})
			}
		}
		for _, custom := range rule.Custom {
			values, ok := fieldValues(bodyMap, custom.Field)
			if !ok {
				continue
			}
			for _, value := range values {
				if err := validateCustomRule(custom.Validator, value); err != nil {
					errs = append(errs, dryRunValidationError{
						Path:      custom.Field,
						Type:      "custom",
						Source:    validationSourceRules,
						Rule:      custom.Validator,
						Message:   err.Error(),
						Condition: condition,
						Actual:    value,
					})
					break
				}
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
	values, ok := fieldValues(body, fieldPath)
	if !ok || len(values) == 0 {
		return false, 0
	}
	if strings.Contains(fieldPath, "[]") {
		for _, value := range values {
			if present, _ := singleValuePresence(value); !present {
				return false, len(values)
			}
		}
		return true, len(values)
	}
	return singleValuePresence(values[0])
}

func singleValuePresence(value any) (bool, int) {
	if value == nil {
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
	values, ok := fieldValues(body, fieldPath)
	if !ok || len(values) == 0 {
		return nil, false
	}
	return values[0], true
}

func fieldValues(body map[string]any, fieldPath string) ([]any, bool) {
	fieldPath = strings.TrimSpace(fieldPath)
	if fieldPath == "." || fieldPath == "$" {
		return []any{body}, true
	}

	current := []any{body}
	for _, rawSegment := range strings.Split(fieldPath, ".") {
		if rawSegment == "" {
			return nil, false
		}
		arrayItems := strings.HasSuffix(rawSegment, "[]")
		segment := strings.TrimSuffix(rawSegment, "[]")
		next := make([]any, 0)
		for _, item := range current {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value, ok := obj[segment]
			if !ok || value == nil {
				continue
			}
			if !arrayItems {
				next = append(next, value)
				continue
			}
			items, ok := value.([]any)
			if !ok {
				continue
			}
			next = append(next, items...)
		}
		if len(next) == 0 {
			return nil, false
		}
		current = next
	}
	return current, true
}

func missingEachField(body map[string]any, fieldPath string) []string {
	segments := strings.Split(strings.TrimSpace(fieldPath), ".")
	if len(segments) < 2 {
		if present, _ := fieldPresence(body, fieldPath); present {
			return nil
		}
		return []string{fieldPath}
	}
	parentPath := strings.Join(segments[:len(segments)-1], ".")
	childPath := segments[len(segments)-1]
	items, ok := fieldValues(body, parentPath)
	if !ok {
		return []string{fieldPath}
	}
	var missing []string
	for idx, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			missing = append(missing, fmt.Sprintf("%s[%d]", parentPath, idx))
			continue
		}
		if present, _ := fieldPresence(obj, childPath); !present {
			missing = append(missing, fmt.Sprintf("%s[%d].%s", parentPath, idx, childPath))
		}
	}
	return missing
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

func validateCustomRule(name string, value any) error {
	switch name {
	case "disabled_string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("Field must be a string.")
		}
		if text != "Disabled" {
			return fmt.Errorf("Field must be \"Disabled\" when this condition is met.")
		}
		return nil
	case "ldap_filter":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("Field must be a string LDAP filter.")
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		if !strings.HasPrefix(text, "(") || !strings.HasSuffix(text, ")") {
			return fmt.Errorf("LDAP filter must start with '(' and end with ')'.")
		}
		if !strings.Contains(text, "=") {
			return fmt.Errorf("LDAP filter must contain an attribute comparison.")
		}
		depth := 0
		for _, r := range text {
			switch r {
			case '(':
				depth++
			case ')':
				depth--
				if depth < 0 {
					return fmt.Errorf("LDAP filter must have balanced parentheses.")
				}
			}
		}
		if depth != 0 {
			return fmt.Errorf("LDAP filter must have balanced parentheses.")
		}
		return nil
	case "native_vlan_in_allowed_vlans":
		settings, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		native, ok := asFloat64(settings["NativeVlan"])
		if !ok {
			return nil
		}
		allowed, ok := settings["AllowedVlans"].(string)
		if !ok || strings.TrimSpace(allowed) == "" {
			return nil
		}
		if vlanInRanges(int(native), allowed) {
			return nil
		}
		return fmt.Errorf("NativeVlan must be included in AllowedVlans.")
	case "netflow_record_type":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("Field must be a string.")
		}
		if strings.TrimSpace(text) == "" || text == "Invalid" {
			return fmt.Errorf("RecordType must be one of \"IPv4\", \"IPv6\", or \"L2\".")
		}
		return nil
	case "netflow_key_fields":
		settings, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		if anyTrueBoolean(settings) {
			return nil
		}
		return fmt.Errorf("At least one key field must be enabled.")
	case "netflow_non_key_fields":
		settings, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		if anyTrueBoolean(settings) {
			return nil
		}
		return fmt.Errorf("At least one non-key field must be enabled.")
	case "ippool_ipv4_blocks_require_config":
		body, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		blocks, ok := body["IpV4Blocks"].([]any)
		if !ok || len(blocks) == 0 {
			return nil
		}
		if ipv4ConfigHasNetmask(body["IpV4Config"]) {
			return nil
		}
		for _, block := range blocks {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			if ipv4ConfigHasNetmask(blockMap["IpV4Config"]) {
				return nil
			}
		}
		return fmt.Errorf("IPv4 pools require IpV4Config.Netmask at the pool level or on each IPv4 block.")
	case "persistent_memory_os_mode":
		body, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		mode, _ := body["ManagementMode"].(string)
		if mode != "configured-from-operating-system" {
			return nil
		}
		for _, field := range []string{"Goals", "LocalSecurity", "LogicalNamespaces", "RetainNamespaces"} {
			if present, _ := fieldPresence(body, field); present {
				return fmt.Errorf("%s must not be provided when ManagementMode is configured-from-operating-system.", field)
			}
		}
		return nil
	case "iqnpool_suffix_blocks_require_suffix":
		body, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("Field must be an object.")
		}
		blocks, ok := body["IqnSuffixBlocks"].([]any)
		if !ok || len(blocks) == 0 {
			return nil
		}
		for _, block := range blocks {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			suffix, _ := blockMap["Suffix"].(string)
			if strings.TrimSpace(suffix) == "" {
				return fmt.Errorf("Each IqnSuffixBlocks entry must include a non-empty Suffix.")
			}
		}
		return nil
	default:
		return fmt.Errorf("Unsupported semantic validator %q.", name)
	}
}

func ipv4ConfigHasNetmask(value any) bool {
	config, ok := value.(map[string]any)
	if !ok {
		return false
	}
	netmask, _ := config["Netmask"].(string)
	return strings.TrimSpace(netmask) != ""
}

func anyTrueBoolean(settings map[string]any) bool {
	for _, value := range settings {
		enabled, ok := value.(bool)
		if ok && enabled {
			return true
		}
	}
	return false
}

func vlanInRanges(vlan int, allowed string) bool {
	for _, part := range strings.Split(allowed, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				continue
			}
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				continue
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				continue
			}
			if start <= vlan && vlan <= end {
				return true
			}
			continue
		}
		value, err := strconv.Atoi(part)
		if err == nil && value == vlan {
			return true
		}
	}
	return false
}
