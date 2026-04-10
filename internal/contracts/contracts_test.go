package contracts

import (
	"errors"
	"testing"
)

func TestNormalizeErrorAuth(t *testing.T) {
	t.Parallel()

	got := NormalizeError(AuthError{Message: "token refresh failed"}, []string{"log1"})
	if got.OK {
		t.Fatalf("expected error envelope")
	}
	if got.Error.Type != ErrorTypeAuth {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	if !got.Error.Retryable {
		t.Fatalf("expected auth error to be retryable")
	}
	if got.Error.Hint != "Check INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET." {
		t.Fatalf("unexpected hint: %q", got.Error.Hint)
	}
	if len(got.Logs) != 1 || got.Logs[0] != "log1" {
		t.Fatalf("unexpected logs: %#v", got.Logs)
	}
}

func TestNormalizeErrorHTTP(t *testing.T) {
	t.Parallel()

	got := NormalizeError(HTTPError{Status: 503, Body: map[string]any{"message": "down"}}, nil)
	if got.Error.Type != ErrorTypeHTTP {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	if got.Error.Status == nil || *got.Error.Status != 503 {
		t.Fatalf("unexpected status: %#v", got.Error.Status)
	}
	if !got.Error.Retryable {
		t.Fatalf("expected 503 to be retryable")
	}
	if got.Error.Details == nil {
		t.Fatalf("expected details to be preserved")
	}
}

func TestNormalizeErrorReferenceHint(t *testing.T) {
	t.Parallel()

	got := NormalizeError(ReferenceError{Message: "api is not defined"}, nil)
	if got.Error.Type != ErrorTypeReference {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	want := "The public runtime no longer exposes api. Use sdk for execution, and use search with catalog, sdk, rules, or spec for discovery."
	if got.Error.Hint != want {
		t.Fatalf("unexpected hint: %q", got.Error.Hint)
	}
}

func TestNormalizeErrorReferenceHintFromWrappedError(t *testing.T) {
	t.Parallel()

	got := NormalizeError(ReferenceError{Err: errors.New("spec is not defined")}, nil)
	if got.Error.Type != ErrorTypeReference {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	want := "The query and mutate tools do not expose spec. Use search to inspect the spec."
	if got.Error.Hint != want {
		t.Fatalf("unexpected hint: %q", got.Error.Hint)
	}
}

func TestNormalizeErrorOutputTooLarge(t *testing.T) {
	t.Parallel()

	got := NormalizeError(OutputTooLarge{Message: "too large", Details: map[string]any{"bytes": 99}}, nil)
	if got.Error.Type != ErrorTypeOutputTooBig {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	if !got.Error.Retryable {
		t.Fatalf("expected oversized output to be retryable")
	}
	if got.Error.Details == nil {
		t.Fatalf("expected details to be preserved")
	}
}

func TestNormalizeErrorWrappedInternalFallback(t *testing.T) {
	t.Parallel()

	got := NormalizeError(errors.New("boom"), nil)
	if got.Error.Type != ErrorTypeInternal {
		t.Fatalf("unexpected type: %q", got.Error.Type)
	}
	if got.Error.Message != "boom" {
		t.Fatalf("unexpected message: %q", got.Error.Message)
	}
}
