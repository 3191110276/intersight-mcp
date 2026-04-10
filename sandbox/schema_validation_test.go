package sandbox

import "testing"

func TestValidateRequestBodyAgainstSchemaRequiredIssue(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
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

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type:                 "object",
		Properties:           map[string]*dryRunSchema{},
		AdditionalProperties: []byte("false"),
	}, map[string]any{"Extra": true})

	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$.Extra", "unknown_field", validationSourceOpenAPI)
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

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
		Type: "string",
		Enum: []any{"fast", "safe"},
	}, "turbo")
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1", len(errs))
	}
	assertDryRunIssue(t, errs[0], "$", "enum", validationSourceOpenAPI)
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

func TestValidateRequestBodyAgainstSchemaRelationshipIssues(t *testing.T) {
	t.Parallel()

	errs := validateRequestBodyAgainstSchema(&dryRunSpecIndex{}, &dryRunSchema{
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

	normalized := normalizeValueForSchema(&dryRunSpecIndex{}, &dryRunSchema{
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

	normalized := normalizeValueForSchema(&dryRunSpecIndex{}, &dryRunSchema{
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
