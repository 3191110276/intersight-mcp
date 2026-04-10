package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

const dryRunPageSize = 1000

type dryRunResult struct {
	DryRun            bool                    `json:"dryRun"`
	Method            string                  `json:"method"`
	Path              string                  `json:"path"`
	Valid             bool                    `json:"valid"`
	Checks            []dryRunCheck           `json:"checks"`
	ValidationErrors  []dryRunValidationError `json:"validationErrors,omitempty"`
	MissingReferences []dryRunReference       `json:"missingReferences"`
	Warnings          []string                `json:"warnings"`
	PredictedRequest  dryRunPredictedRequest  `json:"predictedRequest"`
	APICallsUsed      int                     `json:"apiCallsUsed"`
}

type dryRunCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type dryRunReference struct {
	Field      string `json:"field"`
	ObjectType string `json:"objectType,omitempty"`
	Moid       string `json:"moid,omitempty"`
	Path       string `json:"path,omitempty"`
	Reason     string `json:"reason"`
}

type dryRunPredictedRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query,omitempty"`
	Body   any               `json:"body,omitempty"`
}

type deleteDependencyRule struct {
	Endpoint    string
	RelationKey string
	Label       string
}

var deleteDependencyRules = map[string][]deleteDependencyRule{
	"/api/v1/vnic/LanConnectivityPolicies": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "LanConnectivityPolicy", Label: "attached Ethernet interfaces"},
	},
	"/api/v1/macpool/Pools": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "MacPool", Label: "Ethernet interfaces using this MAC pool"},
	},
	"/api/v1/fabric/EthNetworkGroupPolicies": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "FabricEthNetworkGroupPolicy", Label: "Ethernet interfaces using this Ethernet Network Group Policy"},
	},
	"/api/v1/vnic/EthNetworkPolicies": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthNetworkPolicy", Label: "Ethernet interfaces using this Ethernet Network Policy"},
	},
	"/api/v1/vnic/EthQosPolicies": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthQosPolicy", Label: "Ethernet interfaces using this Ethernet QoS Policy"},
	},
	"/api/v1/vnic/EthAdapterPolicies": {
		{Endpoint: "/api/v1/vnic/EthIfs", RelationKey: "EthAdapterPolicy", Label: "Ethernet interfaces using this Ethernet Adapter Policy"},
	},
}

type moRefCandidate struct {
	Field      string
	ObjectType string
	Moid       string
}

func (b *apiBridge) executeDryRun(execCtx context.Context, method, requestPath string, options APIRequestOptions) (any, error) {
	result := dryRunResult{
		DryRun: true,
		Method: method,
		Path:   requestPath,
		Valid:  true,
		PredictedRequest: dryRunPredictedRequest{
			Method: method,
			Path:   requestPath,
			Query:  cloneStringMap(options.Query),
			Body:   options.Body,
		},
		Warnings: []string{
			"Best-effort dry-run only. Results are based on local validation and read-only GET checks.",
		},
	}

	addCheck := func(name, status, message string) {
		result.Checks = append(result.Checks, dryRunCheck{Name: name, Status: status, Message: message})
		if status == "fail" {
			result.Valid = false
		}
	}

	if strings.TrimSpace(requestPath) == "" {
		addCheck("path", "fail", "Request path must be non-empty.")
		result.APICallsUsed = b.APICallCount()
		return result, nil
	}

	switch method {
	case http.MethodPost, http.MethodPatch:
		if options.Body == nil {
			addCheck("body", "fail", fmt.Sprintf("%s dry-runs require a JSON body.", method))
			result.APICallsUsed = b.APICallCount()
			return result, nil
		}
		addCheck("body", "pass", "Request body is present.")
	case http.MethodDelete:
		if options.Body != nil {
			result.Warnings = append(result.Warnings, "DELETE dry-run ignores the request body.")
			addCheck("body", "warn", "DELETE dry-run ignores the request body.")
		}
	default:
		addCheck("method", "fail", "Dry-run only supports POST, PATCH, and DELETE preview flows.")
		result.APICallsUsed = b.APICallCount()
		return result, nil
	}

	if method == http.MethodPatch {
		if ok, message := validatePathBodyMoid(requestPath, options.Body); ok {
			addCheck("path-body-moid", "pass", message)
		} else {
			addCheck("path-body-moid", "fail", message)
		}
	}

	if method == http.MethodDelete {
		if err := b.checkDelete(execCtx, requestPath, &result); err != nil {
			return nil, err
		}
		result.APICallsUsed = b.APICallCount()
		return result, nil
	}

	refs := collectMoRefs(options.Body, "")

	requestSchema := b.requestSchema(method, requestPath)
	switch {
	case b.spec == nil:
		addCheck("schema", "warn", "Embedded request schema is unavailable, so payload validation was skipped.")
		result.Warnings = append(result.Warnings, "Spec-derived payload validation was skipped because no embedded spec is loaded in this executor.")
	case requestSchema == nil:
		addCheck("schema", "warn", "Could not resolve an application/json request schema for this operation.")
		result.Warnings = append(result.Warnings, "Spec-derived payload validation was skipped because no request schema matched the method and path.")
	default:
		validationErrors := validateRequestBodyAgainstSchema(b.spec, requestSchema, options.Body)
		if len(validationErrors) > 0 {
			result.ValidationErrors = validationErrors
			addCheck("schema", "fail", fmt.Sprintf("Request body failed %d schema validation check(s).", len(validationErrors)))
			result.Warnings = append(result.Warnings, "Reference checks were skipped because the request body failed spec-derived schema validation.")
			result.APICallsUsed = b.APICallCount()
			return result, nil
		}
		addCheck("schema", "pass", "Request body passed spec-derived schema validation.")
	}

	if len(refs) == 0 {
		addCheck("references", "warn", "No resolvable MoRef associations were found in the request body.")
		result.Warnings = append(result.Warnings, "Service-side validation may still require associations not detectable from the submitted body.")
		result.APICallsUsed = b.APICallCount()
		return result, nil
	}

	for _, ref := range refs {
		if strings.TrimSpace(ref.ObjectType) == "" || strings.TrimSpace(ref.Moid) == "" {
			reason := "MoRef is missing ObjectType or Moid."
			addCheck("reference:"+ref.Field, "fail", reason)
			result.MissingReferences = append(result.MissingReferences, dryRunReference{
				Field:      ref.Field,
				ObjectType: ref.ObjectType,
				Moid:       ref.Moid,
				Reason:     reason,
			})
			continue
		}

		refPath, ok := objectTypeToPath(ref.ObjectType)
		if !ok {
			addCheck("reference:"+ref.Field, "warn", "Unable to resolve an API path for this MoRef object type.")
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not resolve object type %q for field %q.", ref.ObjectType, ref.Field))
			continue
		}

		targetPath := strings.TrimRight(refPath, "/") + "/" + ref.Moid
		exists, err := b.resourceExists(execCtx, targetPath)
		if err != nil {
			return nil, err
		}
		if exists {
			addCheck("reference:"+ref.Field, "pass", fmt.Sprintf("Resolved %s %s.", ref.ObjectType, ref.Moid))
			continue
		}

		reason := "Referenced object was not found."
		addCheck("reference:"+ref.Field, "fail", reason)
		result.MissingReferences = append(result.MissingReferences, dryRunReference{
			Field:      ref.Field,
			ObjectType: ref.ObjectType,
			Moid:       ref.Moid,
			Path:       targetPath,
			Reason:     reason,
		})
	}

	if result.Valid {
		addCheck("summary", "pass", "All resolvable references passed read-only checks.")
	}
	result.APICallsUsed = b.APICallCount()
	return result, nil
}

func (b *apiBridge) checkDelete(execCtx context.Context, requestPath string, result *dryRunResult) error {
	exists, err := b.resourceExists(execCtx, requestPath)
	if err != nil {
		return err
	}
	if !exists {
		result.Valid = false
		result.Checks = append(result.Checks, dryRunCheck{
			Name:    "target",
			Status:  "fail",
			Message: "Delete target was not found.",
		})
		result.MissingReferences = append(result.MissingReferences, dryRunReference{
			Field:  "target",
			Path:   requestPath,
			Reason: "Delete target was not found.",
		})
		return nil
	}

	result.Checks = append(result.Checks, dryRunCheck{
		Name:    "target",
		Status:  "pass",
		Message: "Delete target exists.",
	})

	basePath, moid := splitCollectionPath(requestPath)
	rules := deleteDependencyRules[basePath]
	if len(rules) == 0 {
		result.Checks = append(result.Checks, dryRunCheck{
			Name:    "dependencies",
			Status:  "warn",
			Message: "No direct dependency rules are defined for this object type.",
		})
		result.Warnings = append(result.Warnings, "No direct dependency checks were available for this delete target.")
		return nil
	}

	for _, rule := range rules {
		dependents, err := b.findDirectDependents(execCtx, rule, moid)
		if err != nil {
			return err
		}
		if len(dependents) == 0 {
			result.Checks = append(result.Checks, dryRunCheck{
				Name:    "dependency:" + rule.Endpoint,
				Status:  "pass",
				Message: "No direct dependents were found.",
			})
			continue
		}
		result.Valid = false
		result.Checks = append(result.Checks, dryRunCheck{
			Name:    "dependency:" + rule.Endpoint,
			Status:  "fail",
			Message: fmt.Sprintf("Found %d direct dependent object(s): %s.", len(dependents), strings.Join(dependents, ", ")),
		})
	}

	if result.Valid {
		result.Warnings = append(result.Warnings, "No direct dependents were found. Intersight may still reject the delete for reasons this preflight cannot detect.")
	}
	return nil
}

func (b *apiBridge) resourceExists(execCtx context.Context, requestPath string) (bool, error) {
	_, err := b.doReadOnly(execCtx, requestPath, nil)
	if err == nil {
		return true, nil
	}
	var httpErr contracts.HTTPError
	if errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

func (b *apiBridge) findDirectDependents(execCtx context.Context, rule deleteDependencyRule, targetMoid string) ([]string, error) {
	var names []string
	skip := 0
	for {
		page, err := b.doReadOnly(execCtx, rule.Endpoint, map[string]string{
			"$top":    fmt.Sprintf("%d", dryRunPageSize),
			"$skip":   fmt.Sprintf("%d", skip),
			"$select": "Name,Moid," + rule.RelationKey,
		})
		if err != nil {
			return nil, err
		}

		payload, _ := page.(map[string]any)
		results, _ := payload["Results"].([]any)
		for _, raw := range results {
			item, _ := raw.(map[string]any)
			if referencesTarget(item[rule.RelationKey], targetMoid) {
				name := stringValue(item["Name"])
				if name == "" {
					name = stringValue(item["Moid"])
				}
				if name == "" {
					name = "<unnamed>"
				}
				names = append(names, name)
			}
		}

		if len(results) < dryRunPageSize {
			break
		}
		skip += dryRunPageSize
	}
	return names, nil
}

func (b *apiBridge) doReadOnly(execCtx context.Context, requestPath string, query map[string]string) (any, error) {
	count := int(b.callCount.Add(1))
	if count > b.maxAPICalls {
		return nil, contracts.LimitError{
			Message: fmt.Sprintf("API call limit reached (%d/%d)", b.maxAPICalls, b.maxAPICalls),
		}
	}

	callCtx, cancel := context.WithTimeout(execCtx, b.perCallTimeout)
	defer cancel()

	result, err := b.client.Do(callCtx, compileOperation(http.MethodGet, requestPath, APIRequestOptions{Query: query}))
	if err != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return nil, contracts.TimeoutError{Message: "execution timed out"}
		}
		if errors.Is(execCtx.Err(), context.Canceled) {
			return nil, contracts.InternalError{Message: "execution canceled", Err: execCtx.Err()}
		}
		return nil, err
	}
	return result, nil
}

func collectMoRefs(value any, field string) []moRefCandidate {
	switch typed := value.(type) {
	case map[string]any:
		if objectType, hasObjectType := typed["ObjectType"]; hasObjectType {
			if moid, hasMoid := typed["Moid"]; hasMoid {
				return []moRefCandidate{{
					Field:      fieldOrRoot(field),
					ObjectType: stringValue(objectType),
					Moid:       stringValue(moid),
				}}
			}
		}
		var refs []moRefCandidate
		for key, child := range typed {
			childField := key
			if field != "" {
				childField = field + "." + key
			}
			refs = append(refs, collectMoRefs(child, childField)...)
		}
		return refs
	case []any:
		var refs []moRefCandidate
		for i, child := range typed {
			refs = append(refs, collectMoRefs(child, fmt.Sprintf("%s[%d]", fieldOrRoot(field), i))...)
		}
		return refs
	default:
		return nil
	}
}

func validatePathBodyMoid(requestPath string, body any) (bool, string) {
	bodyMap, ok := body.(map[string]any)
	if !ok {
		return true, "Request body has no top-level Moid to compare."
	}
	bodyMoid := strings.TrimSpace(stringValue(bodyMap["Moid"]))
	if bodyMoid == "" {
		return true, "Request body has no top-level Moid to compare."
	}
	_, pathMoid := splitCollectionPath(requestPath)
	if pathMoid == "" {
		return true, "Request path has no Moid to compare."
	}
	if bodyMoid == pathMoid {
		return true, "Request path and body Moid match."
	}
	return false, fmt.Sprintf("Request path Moid %q does not match body Moid %q.", pathMoid, bodyMoid)
}

func splitCollectionPath(requestPath string) (string, string) {
	clean := path.Clean("/" + strings.TrimSpace(requestPath))
	if clean == "." || clean == "/" {
		return "", ""
	}
	if !strings.HasPrefix(clean, "/api/v1/") {
		return clean, ""
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	if len(parts) < 4 {
		return clean, ""
	}
	return "/" + strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]
}

func referencesTarget(value any, targetMoid string) bool {
	switch typed := value.(type) {
	case map[string]any:
		return strings.TrimSpace(stringValue(typed["Moid"])) == targetMoid
	case []any:
		for _, item := range typed {
			if referencesTarget(item, targetMoid) {
				return true
			}
		}
	}
	return false
}

func objectTypeToPath(objectType string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(objectType), ".")
	if len(parts) != 2 {
		return "", false
	}
	return "/api/v1/" + parts[0] + "/" + pluralizeType(parts[1]), true
}

func pluralizeType(name string) string {
	switch {
	case strings.HasSuffix(name, "y") && len(name) > 1 && !strings.ContainsRune("aeiouAEIOU", rune(name[len(name)-2])):
		return name[:len(name)-1] + "ies"
	case strings.HasSuffix(name, "s"):
		return name + "es"
	default:
		return name + "s"
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func fieldOrRoot(field string) string {
	if field == "" {
		return "$"
	}
	return field
}

func (b *apiBridge) requestSchema(method, requestPath string) *dryRunSchema {
	if b.spec == nil {
		return nil
	}
	return b.spec.requestSchema(method, requestPath)
}
