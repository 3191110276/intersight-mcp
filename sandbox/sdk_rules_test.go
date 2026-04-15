package sandbox

import (
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestValidateSemanticRulesTypedIssues(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Require: []contracts.FieldRule{{Field: "Organization"}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Require: []contracts.FieldRule{{Field: "Tags", MinCount: 2}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Forbid: []string{"Deprecated"},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Minimum: []contracts.MinimumRule{{Field: "Priority", Value: 10}},
						},
						{
							Kind:       "conditional",
							When:       &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							RequireAny: []contracts.FieldRule{{Field: "Primary"}, {Field: "Secondary"}},
						},
						{
							Kind:        "conditional",
							When:        &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							RequireEach: []contracts.FieldRule{{Field: "Items[].Name"}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Maximum: []contracts.LengthRule{{Field: "Username", Value: 4}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Pattern: []contracts.PatternRule{{Field: "Slug", Value: "^[a-z]+$"}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Future: []contracts.TimeRule{{Field: "StartsAt"}},
						},
						{
							Kind:     "conditional",
							When:     &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Contains: []contracts.ContainsRule{{Field: "Kinds[]", Value: "gpu"}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Custom: []contracts.CustomRule{{Field: "Filter", Validator: "ldap_filter"}},
						},
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Mode":       "fast",
		"Tags":       []any{"one"},
		"Deprecated": true,
		"Priority":   5,
		"Items":      []any{map[string]any{}, map[string]any{"Name": "ok"}},
		"Username":   "too-long",
		"Slug":       "UPPER",
		"StartsAt":   "2020-01-01T00:00:00Z",
		"Kinds":      []any{"cpu"},
		"Filter":     "uid=user",
	})

	if len(errs) != 11 {
		t.Fatalf("len(errs) = %d, want 11", len(errs))
	}
	assertSemanticIssue(t, errs[0], "Organization", "required")
	assertSemanticIssue(t, errs[1], "Tags", "min_items")
	assertSemanticIssue(t, errs[2], "Deprecated", "forbidden")
	assertSemanticIssue(t, errs[3], "Priority", "minimum")
	assertSemanticIssue(t, errs[4], "Primary|Secondary", "one_of")
	assertSemanticIssue(t, errs[5], "Items[].Name", "required_each")
	assertSemanticIssue(t, errs[6], "Username", "maximum")
	assertSemanticIssue(t, errs[7], "Slug", "pattern")
	assertSemanticIssue(t, errs[8], "StartsAt", "future")
	assertSemanticIssue(t, errs[9], "Kinds[]", "contains")
	assertSemanticIssue(t, errs[10], "Filter", "custom")

	for _, err := range errs {
		if err.Condition == "" {
			t.Fatalf("condition = %q, want non-empty", err.Condition)
		}
	}
}

func TestValidateSemanticRulesOneOfSatisfied(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						{
							Kind:       "one_of",
							RequireAny: []contracts.FieldRule{{Field: "Primary"}, {Field: "Secondary"}},
						},
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Secondary": "value",
	})
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestValidateSemanticRulesCustomProbeValidators(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewConditionalInCustomRule("Mode", []any{"auto", "on"}, contracts.CustomRule{Field: "ReceiveDirection", Validator: "disabled_string"}),
						contracts.NewCustomRule(contracts.CustomRule{Field: "VlanSettings", Validator: "native_vlan_in_allowed_vlans"}),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Mode":             "auto",
		"ReceiveDirection": "Enabled",
		"VlanSettings": map[string]any{
			"AllowedVlans": "2-3",
			"NativeVlan":   1,
		},
	})

	if len(errs) != 2 {
		t.Fatalf("len(errs) = %d, want 2: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "ReceiveDirection", "custom")
	assertSemanticIssue(t, errs[1], "VlanSettings", "custom")
}

func assertSemanticIssue(t *testing.T, err dryRunValidationError, path, issueType string) {
	t.Helper()

	if err.Path != path {
		t.Fatalf("path = %q, want %q", err.Path, path)
	}
	if err.Type != issueType {
		t.Fatalf("type = %q, want %q", err.Type, issueType)
	}
	if err.Source != validationSourceRules {
		t.Fatalf("source = %q, want %q", err.Source, validationSourceRules)
	}
}
