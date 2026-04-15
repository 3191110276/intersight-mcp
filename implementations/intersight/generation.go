package intersight

import (
	"fmt"
	"strings"

	"github.com/mimaurer/intersight-mcp/implementations"
	"github.com/mimaurer/intersight-mcp/internal/contracts"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
)

func SchemaNormalizationHook() implementations.SchemaNormalizationHook {
	return func(proxy *highbase.SchemaProxy, schema *highbase.Schema, out *contracts.NormalizedSchema) {
		if out == nil {
			return
		}
		if expandTarget := deriveIntersightExpandTarget(proxy, schema); expandTarget != "" {
			applyIntersightRelationshipSchema(out, expandTarget)
		}
	}
}

func applyIntersightRelationshipSchema(dst *contracts.NormalizedSchema, target string) {
	if dst == nil {
		return
	}
	dst.Type = "object"
	dst.ExpandTarget = target
	dst.Relationship = true
	dst.RelationshipTarget = target
	dst.RelationshipWriteForms = []string{"moidRef", "typedMoRef"}
	dst.Properties = map[string]*contracts.NormalizedSchema{
		"Moid": {Type: "string"},
		"ObjectType": {
			Type: "string",
			Enum: []any{target},
		},
		"ClassId": {
			Type: "string",
			Enum: []any{"mo.MoRef"},
		},
	}
	dst.OneOf = []*contracts.NormalizedSchema{
		{
			Type: "object",
			Properties: map[string]*contracts.NormalizedSchema{
				"Moid": {Type: "string"},
			},
			Required: []string{"Moid"},
		},
		{
			Type: "object",
			Properties: map[string]*contracts.NormalizedSchema{
				"Moid": {Type: "string"},
				"ObjectType": {
					Type: "string",
					Enum: []any{target},
				},
				"ClassId": {
					Type: "string",
					Enum: []any{"mo.MoRef"},
				},
			},
			Required: []string{"Moid", "ObjectType", "ClassId"},
		},
	}
}

func deriveIntersightExpandTarget(proxy *highbase.SchemaProxy, schema *highbase.Schema) string {
	if schema == nil || len(schema.AllOf) == 0 {
		return ""
	}
	if !schemaHasProperty(schema, "Moid") || !schemaHasProperty(schema, "ObjectType") {
		return ""
	}

	target := ""
	foundMoRef := false
	for _, item := range schema.AllOf {
		refName := schemaRefName(item.GetReference())
		if refName == "" {
			continue
		}
		if refName == "mo.MoRef" || schemaInheritsFromRef(item, "mo.MoRef", map[string]struct{}{}) {
			foundMoRef = true
			continue
		}
		if target != "" {
			return ""
		}
		target = refName
	}
	if !foundMoRef || target == "" {
		return ""
	}
	if proxy != nil && schemaRefName(proxy.GetReference()) == target {
		return ""
	}
	return target
}

func schemaHasProperty(schema *highbase.Schema, name string) bool {
	if schema == nil {
		return false
	}
	if schema.Properties != nil && schema.Properties.GetOrZero(name) != nil {
		return true
	}
	for _, item := range schema.AllOf {
		if schemaHasProperty(item.Schema(), name) {
			return true
		}
	}
	return false
}

func schemaInheritsFromRef(proxy *highbase.SchemaProxy, target string, seen map[string]struct{}) bool {
	if proxy == nil {
		return false
	}
	refName := schemaRefName(proxy.GetReference())
	if refName == target {
		return true
	}
	key := refName
	if key == "" {
		key = fmt.Sprintf("%p", proxy)
	}
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}

	schema := proxy.Schema()
	if schema == nil {
		return false
	}
	for _, item := range schema.AllOf {
		if schemaInheritsFromRef(item, target, seen) {
			return true
		}
	}
	return false
}

func schemaRefName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}
