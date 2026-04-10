package contracts

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ErrorTypeAuth         = "AuthError"
	ErrorTypeHTTP         = "HTTPError"
	ErrorTypeNetwork      = "NetworkError"
	ErrorTypeTimeout      = "TimeoutError"
	ErrorTypeLimit        = "LimitError"
	ErrorTypeValidation   = "ValidationError"
	ErrorTypeReference    = "ReferenceError"
	ErrorTypeOutputTooBig = "OutputTooLarge"
	ErrorTypeInternal     = "InternalError"
)

type SuccessEnvelope struct {
	OK     bool     `json:"ok"`
	Result any      `json:"result"`
	Logs   []string `json:"logs"`
}

type ErrorEnvelope struct {
	OK    bool             `json:"ok"`
	Error OutwardToolError `json:"error"`
	Logs  []string         `json:"logs"`
}

type OutwardToolError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Hint      string `json:"hint"`
	Retryable bool   `json:"retryable"`
	Status    *int   `json:"status,omitempty"`
	Details   any    `json:"details,omitempty"`
}

func Success(result any, logs []string) SuccessEnvelope {
	return SuccessEnvelope{
		OK:     true,
		Result: result,
		Logs:   cloneLogs(logs),
	}
}

func NormalizeError(err error, logs []string) ErrorEnvelope {
	if err == nil {
		err = InternalError{Message: "unknown internal error"}
	}

	var normalized OutwardToolError
	switch e := classify(err).(type) {
	case AuthError:
		normalized = OutwardToolError{
			Type:      ErrorTypeAuth,
			Message:   e.message(),
			Hint:      "Check INTERSIGHT_CLIENT_ID and INTERSIGHT_CLIENT_SECRET.",
			Retryable: true,
		}
	case HTTPError:
		normalized = OutwardToolError{
			Type:      ErrorTypeHTTP,
			Message:   e.message(),
			Hint:      "Inspect the HTTP status and response details, then adjust the request.",
			Retryable: e.Status >= 500 || e.Status == 429,
			Status:    intPtr(e.Status),
			Details:   e.Body,
		}
	case NetworkError:
		normalized = OutwardToolError{
			Type:      ErrorTypeNetwork,
			Message:   e.message(),
			Hint:      "Check network connectivity and the configured endpoint, then retry.",
			Retryable: true,
		}
	case TimeoutError:
		normalized = OutwardToolError{
			Type:      ErrorTypeTimeout,
			Message:   e.message(),
			Hint:      "Reduce the amount of work or narrow the request, then retry.",
			Retryable: true,
		}
	case LimitError:
		normalized = OutwardToolError{
			Type:      ErrorTypeLimit,
			Message:   e.message(),
			Hint:      "Retry shortly or reduce parallel tool calls.",
			Retryable: true,
		}
	case ValidationError:
		normalized = OutwardToolError{
			Type:      ErrorTypeValidation,
			Message:   e.message(),
			Hint:      "Fix the submitted input and retry.",
			Retryable: false,
			Details:   e.Details,
		}
	case ReferenceError:
		message := e.message()
		normalized = OutwardToolError{
			Type:      ErrorTypeReference,
			Message:   message,
			Hint:      referenceHint(message),
			Retryable: false,
		}
	case OutputTooLarge:
		normalized = OutwardToolError{
			Type:      ErrorTypeOutputTooBig,
			Message:   e.message(),
			Hint:      "Reduce the result set with $select, $top, or $filter and retry.",
			Retryable: true,
			Details:   e.Details,
		}
	default:
		ie := InternalError{Message: err.Error()}
		if errors.As(err, &ie) {
			normalized = OutwardToolError{
				Type:      ErrorTypeInternal,
				Message:   ie.message(),
				Hint:      "Retry the request. If the problem persists, inspect debug logs.",
				Retryable: false,
			}
		} else {
			normalized = OutwardToolError{
				Type:      ErrorTypeInternal,
				Message:   err.Error(),
				Hint:      "Retry the request. If the problem persists, inspect debug logs.",
				Retryable: false,
			}
		}
	}

	return ErrorEnvelope{
		OK:    false,
		Error: normalized,
		Logs:  cloneLogs(logs),
	}
}

type AuthError struct {
	Message string
	Err     error
}

func (e AuthError) Error() string { return e.message() }
func (e AuthError) Unwrap() error { return e.Err }
func (e AuthError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "authentication failed"
}

type HTTPError struct {
	Status  int
	Body    any
	Message string
	Err     error
}

func (e HTTPError) Error() string { return e.message() }
func (e HTTPError) Unwrap() error { return e.Err }
func (e HTTPError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Status > 0 {
		return fmt.Sprintf("Intersight returned HTTP %d", e.Status)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "HTTP request failed"
}

type NetworkError struct {
	Message string
	Err     error
}

func (e NetworkError) Error() string { return e.message() }
func (e NetworkError) Unwrap() error { return e.Err }
func (e NetworkError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "network request failed"
}

type TimeoutError struct {
	Message string
	Err     error
}

func (e TimeoutError) Error() string { return e.message() }
func (e TimeoutError) Unwrap() error { return e.Err }
func (e TimeoutError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "request timed out"
}

type LimitError struct {
	Message string
	Err     error
}

func (e LimitError) Error() string { return e.message() }
func (e LimitError) Unwrap() error { return e.Err }
func (e LimitError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "execution limit reached"
}

type ValidationError struct {
	Message string
	Details any
	Err     error
}

func (e ValidationError) Error() string { return e.message() }
func (e ValidationError) Unwrap() error { return e.Err }
func (e ValidationError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "validation failed"
}

type ReferenceError struct {
	Message string
	Err     error
}

func (e ReferenceError) Error() string { return e.message() }
func (e ReferenceError) Unwrap() error { return e.Err }
func (e ReferenceError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "reference error"
}

type OutputTooLarge struct {
	Message string
	Details any
	Err     error
}

func (e OutputTooLarge) Error() string { return e.message() }
func (e OutputTooLarge) Unwrap() error { return e.Err }
func (e OutputTooLarge) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "output exceeds configured limit"
}

type InternalError struct {
	Message string
	Err     error
}

func (e InternalError) Error() string { return e.message() }
func (e InternalError) Unwrap() error { return e.Err }
func (e InternalError) message() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "internal server error"
}

func classify(err error) error {
	var authErr AuthError
	if errors.As(err, &authErr) {
		return authErr
	}
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr
	}
	var networkErr NetworkError
	if errors.As(err, &networkErr) {
		return networkErr
	}
	var timeoutErr TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr
	}
	var limitErr LimitError
	if errors.As(err, &limitErr) {
		return limitErr
	}
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		return validationErr
	}
	var referenceErr ReferenceError
	if errors.As(err, &referenceErr) {
		return referenceErr
	}
	var tooLargeErr OutputTooLarge
	if errors.As(err, &tooLargeErr) {
		return tooLargeErr
	}
	var internalErr InternalError
	if errors.As(err, &internalErr) {
		return internalErr
	}
	return err
}

func referenceHint(message string) string {
	switch {
	case strings.Contains(message, "api is not defined"):
		return "The public runtime no longer exposes api. Use sdk for execution, and use search with catalog, sdk, rules, or spec for discovery."
	case strings.Contains(message, "spec is not defined"):
		return "The query and mutate tools do not expose spec. Use search to inspect the spec."
	default:
		return "Use only the globals documented for this tool and retry."
	}
}

func cloneLogs(logs []string) []string {
	if len(logs) == 0 {
		return []string{}
	}
	return append([]string(nil), logs...)
}

func intPtr(v int) *int {
	return &v
}
