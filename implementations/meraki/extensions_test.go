package meraki

import (
	"strings"
	"testing"
)

func TestSandboxExtensionsExposeListAllHelpers(t *testing.T) {
	t.Parallel()

	ext := SandboxExtensions()
	if len(ext.CustomSDKMethods) == 0 {
		t.Fatal("expected custom SDK methods")
	}

	method, ok := ext.CustomSDKMethods["administered.licensingsubscriptionsubscriptions.listAll"]
	if !ok {
		t.Fatal("expected administered.licensingsubscriptionsubscriptions.listAll helper")
	}
	if method.CompileOperation == nil {
		t.Fatal("expected listAll helper to compile operations")
	}
}

func TestCompileMerakiListAllOperationSetsFollowUpPlan(t *testing.T) {
	t.Parallel()

	methods, err := merakiCustomSDKMethods()
	if err != nil {
		t.Fatalf("merakiCustomSDKMethods() error = %v", err)
	}
	helper, ok := methods["administered.licensingsubscriptionsubscriptions.listAll"]
	if !ok {
		t.Fatal("expected helper to be registered")
	}

	op, err := helper.CompileOperation(map[string]any{
		"query": map[string]any{
			"perPage": "50",
		},
	}, "query", false)
	if err != nil {
		t.Fatalf("CompileOperation() error = %v", err)
	}
	if op.FollowUpPlan.Kind != merakiListAllFollowUpKind {
		t.Fatalf("FollowUpPlan.Kind = %q, want %q", op.FollowUpPlan.Kind, merakiListAllFollowUpKind)
	}
	if !strings.HasSuffix(op.PathTemplate, "/subscription/subscriptions") {
		t.Fatalf("unexpected PathTemplate = %q", op.PathTemplate)
	}
	if got := op.QueryParams["perPage"][0]; got != "50" {
		t.Fatalf("perPage = %q, want 50", got)
	}
}
