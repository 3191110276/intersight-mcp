package contracts

import "testing"

func TestNewHTTPOperationDescriptorDefaults(t *testing.T) {
	t.Parallel()

	got := NewHTTPOperationDescriptor(" post ", " /api/v1/example/Widgets ")

	if got.Kind != OperationKindHTTP {
		t.Fatalf("Kind = %q, want %q", got.Kind, OperationKindHTTP)
	}
	if got.Method != "POST" {
		t.Fatalf("Method = %q, want POST", got.Method)
	}
	if got.PathTemplate != "/api/v1/example/Widgets" {
		t.Fatalf("PathTemplate = %q", got.PathTemplate)
	}
	if got.Path != "/api/v1/example/Widgets" {
		t.Fatalf("Path = %q", got.Path)
	}
	if got.ResponseMode != ResponseModeJSON {
		t.Fatalf("ResponseMode = %q, want %q", got.ResponseMode, ResponseModeJSON)
	}
	if got.ValidationPlan.Kind != ValidationPlanNone {
		t.Fatalf("ValidationPlan.Kind = %q, want %q", got.ValidationPlan.Kind, ValidationPlanNone)
	}
	if got.FollowUpPlan.Kind != FollowUpPlanNone {
		t.Fatalf("FollowUpPlan.Kind = %q, want %q", got.FollowUpPlan.Kind, FollowUpPlanNone)
	}
	if got.PathParams == nil || got.QueryParams == nil || got.Headers == nil {
		t.Fatalf("expected initialized maps: %#v", got)
	}
}
