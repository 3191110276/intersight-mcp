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
	})

	if len(errs) != 5 {
		t.Fatalf("len(errs) = %d, want 5", len(errs))
	}
	assertSemanticIssue(t, errs[0], "Organization", "required")
	assertSemanticIssue(t, errs[1], "Tags", "min_items")
	assertSemanticIssue(t, errs[2], "Deprecated", "forbidden")
	assertSemanticIssue(t, errs[3], "Priority", "minimum")
	assertSemanticIssue(t, errs[4], "Primary|Secondary", "one_of")

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
