package implementations_test

import (
	"slices"
	"testing"

	"github.com/mimaurer/intersight-mcp/implementations"
	_ "github.com/mimaurer/intersight-mcp/implementations/all"
)

func TestLookupTargetRegisteredProvider(t *testing.T) {
	t.Parallel()

	target, err := implementations.LookupTarget("intersight")
	if err != nil {
		t.Fatalf("LookupTarget() error = %v", err)
	}
	if got := target.Name(); got != "intersight" {
		t.Fatalf("target.Name() = %q, want intersight", got)
	}
}

func TestRegisteredTargetNamesIncludesIntersight(t *testing.T) {
	t.Parallel()

	names := implementations.RegisteredTargetNames()
	if !slices.Contains(names, "intersight") {
		t.Fatalf("RegisteredTargetNames() = %v, want intersight", names)
	}
	if !slices.Contains(names, "meraki") {
		t.Fatalf("RegisteredTargetNames() = %v, want meraki", names)
	}
}
