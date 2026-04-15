package sandbox

import "github.com/mimaurer/intersight-mcp/internal/providerext"

type Extensions = providerext.Extensions
type CustomSDKMethod = providerext.CustomSDKMethod
type RelationshipBehavior = providerext.RelationshipBehavior
type DeleteDependencyRule = providerext.DeleteDependencyRule

func normalizeExtensions(ext Extensions) Extensions {
	if ext.CustomSDKMethods == nil {
		ext.CustomSDKMethods = map[string]CustomSDKMethod{}
	}
	if ext.DeleteDependencyRules == nil {
		ext.DeleteDependencyRules = map[string][]DeleteDependencyRule{}
	}
	if ext.RelationshipBehavior != nil {
		if ext.RelationshipBehavior.MoidField == "" {
			ext.RelationshipBehavior.MoidField = "Moid"
		}
		if ext.RelationshipBehavior.ClassIDField == "" {
			ext.RelationshipBehavior.ClassIDField = "ClassId"
		}
		if ext.RelationshipBehavior.ObjectTypeField == "" {
			ext.RelationshipBehavior.ObjectTypeField = "ObjectType"
		}
		if ext.RelationshipBehavior.SelectorMessage == "" {
			ext.RelationshipBehavior.SelectorMessage = "Selector-only relationship payloads are not accepted for writes."
		}
		if ext.RelationshipBehavior.AllowMoidRefWriteForm == "" {
			ext.RelationshipBehavior.AllowMoidRefWriteForm = "moidRef"
		}
		if ext.RelationshipBehavior.AllowTypedMoRefWriteForm == "" {
			ext.RelationshipBehavior.AllowTypedMoRefWriteForm = "typedMoRef"
		}
		if ext.RelationshipBehavior.RelationshipRuleName == "" {
			ext.RelationshipBehavior.RelationshipRuleName = "relationship"
		}
		if ext.RelationshipBehavior.MissingMoidMessage == "" {
			ext.RelationshipBehavior.MissingMoidMessage = "Relationship identifier is required."
		}
		if ext.RelationshipBehavior.InvalidPayloadShapeMessage == "" {
			ext.RelationshipBehavior.InvalidPayloadShapeMessage = "Relationship payload shape is not accepted for writes."
		}
	}
	return ext
}
