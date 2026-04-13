package tools

import "strings"

var apiObjectExcludedFields = map[string]struct{}{
	"AccountMoid":         {},
	"ApplianceAccount":    {},
	"ClassId":             {},
	"CreateTime":          {},
	"DomainGroupMoid":     {},
	"ModTime":             {},
	"Owners":              {},
	"PermissionResources": {},
	"SharedScope":         {},
	"link":                {},
}

func compactToolResult(tool string, value any, compact bool) any {
	if !compact {
		return value
	}
	switch tool {
	case ToolQuery, ToolMutate:
		return compactAPIValue(value)
	default:
		return value
	}
}

func compactAPIValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return compactAPIObject(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = compactAPIValue(typed[i])
		}
		return out
	default:
		return value
	}
}

func compactAPIObject(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(in))
	for key, raw := range in {
		if _, drop := apiObjectExcludedFields[strings.TrimSpace(key)]; drop {
			continue
		}
		out[key] = compactAPIValue(raw)
	}
	return out
}
