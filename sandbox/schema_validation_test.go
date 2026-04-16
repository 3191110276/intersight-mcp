package sandbox

import (
	"testing"

	targetintersight "github.com/mimaurer/intersight-mcp/implementations/intersight"
)

func TestValidateRequestBodyAgainstSchemaRequiredIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type:     "object",
		Required: []string{"Name"},
		Properties: map[string]*dryRunSchema{
			"Name": {Type: "string"},
		},
	}, map[string]any{})

	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$.Name", "required", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaAdditionalPropertyIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type:                 "object",
		Properties:           map[string]*dryRunSchema{},
		AdditionalProperties: []byte("false"),
	}, map[string]any{"Extra": true})

	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$.Extra", "unknown_field", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaImplicitAdditionalPropertyIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type: "object",
		Properties: map[string]*dryRunSchema{
			"Key": {Type: "string"},
		},
	}, map[string]any{"Key": "path", "Value": ""})

	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1: %#v", len(errs), errs)
	}
	assertDryRunIssue(t, errs[0], "$.Value", "unknown_field", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaTypeMismatchIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{Type: "string"}, true)
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$", "type_mismatch", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaEnumIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type: "string",
		Enum: []any{"fast", "safe"},
	}, "turbo")
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$", "enum", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaPatternIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type:    "string",
		Pattern: "^[a-z]+$",
	}, "ABC123")
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$", "pattern", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaPatternPasses(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type:    "string",
		Pattern: "^[a-z]+$",
	}, "alpha")
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestValidateRequestBodyAgainstSchemaIntegerEnumAcceptsNumericTypes(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type: "integer",
		Enum: []any{float64(9600), float64(115200)},
	}, 115200)
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestValidateRequestBodyAgainstSchemaOneOfAndAnyOfIssues(t *testing.T) {
	t.Parallel()

	oneOfErrs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		OneOf: []*dryRunSchema{{Type: "string"}, {Type: "integer"}},
	}, true)
	if len(oneOfErrs) != 1 {
		t.Fatalf("len(oneOfErrs) = %d, want 1", len(oneOfErrs))
	}
	assertDryRunIssue(t, oneOfErrs[0], "$", "one_of", validationSourceOpenAPI)

	anyOfErrs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		AnyOf: []*dryRunSchema{{Type: "string"}, {Type: "integer"}},
	}, true)
	if len(anyOfErrs) != 1 {
		t.Fatalf("len(anyOfErrs) = %d, want 1", len(anyOfErrs))
	}
	assertDryRunIssue(t, anyOfErrs[0], "$", "any_of", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaOneOfIgnoresGenericObjectCatchall(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		OneOf: []*dryRunSchema{
			{
				Type: "object",
				Properties: map[string]*dryRunSchema{
					"Moid": {Type: "string"},
				},
				Required: []string{"Moid"},
			},
			{Type: "object"},
		},
	}, map[string]any{"Moid": "org-1"})

	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestValidateRequestBodyAgainstSchemaRelationshipIssues(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type:                   "object",
		Relationship:           true,
		RelationshipTarget:     "organization.Organization",
		RelationshipWriteForms: []string{"moidRef", "typedMoRef"},
	}, map[string]any{"Selector": "Name eq 'default'"})
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$", "relationship", validationSourceOpenAPI)
}

func TestValidateRequestBodyAgainstSchemaRelationshipAllowsMoidOnly(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type:                   "object",
		Relationship:           true,
		RelationshipTarget:     "organization.Organization",
		RelationshipWriteForms: []string{"moidRef", "typedMoRef"},
	}, map[string]any{"Moid": "org-1"})
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestNormalizeValueForSchemaAddsTopLevelDiscriminators(t *testing.T) {
	t.Parallel()

	normalized := normalizeValueForSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type: "object",
		Properties: map[string]*dryRunSchema{
			"ClassId": {
				Type: "string",
				Enum: []any{"ntp.Policy"},
			},
			"ObjectType": {
				Type: "string",
				Enum: []any{"ntp.Policy"},
			},
			"Name": {Type: "string"},
		},
	}, map[string]any{"Name": "policy-a"}, &schemaValidationState{visiting: map[string]int{}})

	body, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("normalized body type = %T, want map[string]any", normalized)
	}
	if body["ClassId"] != "ntp.Policy" {
		t.Fatalf("ClassId = %#v, want ntp.Policy", body["ClassId"])
	}
	if body["ObjectType"] != "ntp.Policy" {
		t.Fatalf("ObjectType = %#v, want ntp.Policy", body["ObjectType"])
	}
}

func TestNormalizeValueForSchemaDoesNotOverrideExplicitDiscriminators(t *testing.T) {
	t.Parallel()

	normalized := normalizeValueForSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type: "object",
		Properties: map[string]*dryRunSchema{
			"ClassId": {
				Type: "string",
				Enum: []any{"ntp.Policy"},
			},
			"ObjectType": {
				Type: "string",
				Enum: []any{"ntp.Policy"},
			},
		},
	}, map[string]any{
		"ClassId":    "custom.Class",
		"ObjectType": "custom.Object",
	}, &schemaValidationState{visiting: map[string]int{}})

	body, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("normalized body type = %T, want map[string]any", normalized)
	}
	if body["ClassId"] != "custom.Class" {
		t.Fatalf("ClassId = %#v, want custom.Class", body["ClassId"])
	}
	if body["ObjectType"] != "custom.Object" {
		t.Fatalf("ObjectType = %#v, want custom.Object", body["ObjectType"])
	}
}

func TestValidateRequestBodyAgainstSchemaRelaxesMissingPolymorphicDiscriminators(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{ext: targetintersight.SandboxExtensions()}, &dryRunSchema{
		Type:     "object",
		Required: []string{"ClassId", "ObjectType", "FailureThreshold"},
		Properties: map[string]*dryRunSchema{
			"ClassId": {
				Type: "string",
				Enum: []any{"workload.BatchDeployment", "workload.CanaryDeployment"},
			},
			"ObjectType": {
				Type: "string",
				Enum: []any{"workload.BatchDeployment", "workload.CanaryDeployment"},
			},
			"FailureThreshold": {Type: "integer"},
		},
	}, map[string]any{"FailureThreshold": 1})

	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func assertDryRunIssue(t *testing.T, err dryRunValidationError, path, issueType, source string) {
	t.Helper()

	if err.Path != path {
		t.Fatalf("path = %q, want %q", err.Path, path)
	}
	if err.Type != issueType {
		t.Fatalf("type = %q, want %q", err.Type, issueType)
	}
	if err.Source != source {
		t.Fatalf("source = %q, want %q", err.Source, source)
	}
}
