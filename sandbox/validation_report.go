package sandbox

type validationIssue struct {
	Path      string `json:"path,omitempty"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Message   string `json:"message"`
	Rule      string `json:"rule,omitempty"`
	Condition string `json:"condition,omitempty"`
	Expected  any    `json:"expected,omitempty"`
	Actual    any    `json:"actual,omitempty"`
	SDKMethod string `json:"sdkMethod,omitempty"`
}

type validationLayer struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Ran    bool   `json:"ran"`
	Passed bool   `json:"passed"`
}

const (
	validationSourceSDKContract = "sdk_contract"
	validationSourceOpenAPI     = "openapi"
	validationSourceRules       = "rules"
)

func defaultValidationLayers(includeBodyChecks bool) []validationLayer {
	return []validationLayer{
		{Name: "sdk_contract", Source: validationSourceSDKContract, Ran: true, Passed: true},
		{Name: "openapi_request_schema", Source: validationSourceOpenAPI, Ran: includeBodyChecks, Passed: true},
		{Name: "rules_semantic", Source: validationSourceRules, Ran: includeBodyChecks, Passed: true},
	}
}

func validationReportOperation(operation map[string]any) map[string]any {
	return operation
}

