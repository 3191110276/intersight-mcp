package contracts

import "strings"

const (
	OperationKindHTTP = "http-operation"

	ResponseModeJSON = "json"

	ValidationPlanNone        = "none"
	ValidationPlanQueryDryRun = "query-dry-run"
	ValidationPlanValidate    = "validate"

	FollowUpPlanNone = "none"
)

type ValidationPlan struct {
	Kind string `json:"kind"`
}

type FollowUpPlan struct {
	Kind string `json:"kind"`
}

type OperationDescriptor struct {
	Kind           string              `json:"kind"`
	OperationID    string              `json:"operationId,omitempty"`
	Method         string              `json:"method"`
	PathTemplate   string              `json:"pathTemplate"`
	Path           string              `json:"path"`
	PathParams     map[string]string   `json:"pathParams,omitempty"`
	QueryParams    map[string][]string `json:"queryParams,omitempty"`
	Headers        map[string][]string `json:"headers,omitempty"`
	Body           any                 `json:"body,omitempty"`
	ResponseMode   string              `json:"responseMode"`
	ValidationPlan ValidationPlan      `json:"validationPlan"`
	FollowUpPlan   FollowUpPlan        `json:"followUpPlan"`
	EndpointURL    string              `json:"endpointUrl,omitempty"`
}

func NewHTTPOperationDescriptor(method, path string) OperationDescriptor {
	return OperationDescriptor{
		Kind:           OperationKindHTTP,
		Method:         strings.ToUpper(strings.TrimSpace(method)),
		PathTemplate:   strings.TrimSpace(path),
		Path:           strings.TrimSpace(path),
		PathParams:     map[string]string{},
		QueryParams:    map[string][]string{},
		Headers:        map[string][]string{},
		ResponseMode:   ResponseModeJSON,
		ValidationPlan: ValidationPlan{Kind: ValidationPlanNone},
		FollowUpPlan:   FollowUpPlan{Kind: FollowUpPlanNone},
	}
}
