package providerext

import "github.com/mimaurer/intersight-mcp/internal/contracts"

type Extensions struct {
	CustomSDKMethods         map[string]CustomSDKMethod
	AutofillDiscriminators   bool
	RelationshipBehavior     *RelationshipBehavior
	RelationshipPathResolver func(string) (string, bool)
	DeleteDependencyRules    map[string][]DeleteDependencyRule
}

type CustomSDKMethod struct {
	CompileOperation func(args map[string]any, mode string, enableMetricsApps bool) (contracts.OperationDescriptor, error)
	PresentationHint func(sdkMethod string, args map[string]any, enableMetricsApps bool) *PresentationHint
}

type PresentationHint struct {
	Kind string
}

type RelationshipBehavior struct {
	RejectSelector             bool
	SelectorMessage            string
	MoidField                  string
	ClassIDField               string
	ObjectTypeField            string
	DefaultClassID             string
	RequiredClassID            string
	AutofillTargetObjectType   bool
	AllowMoidRefWriteForm      string
	AllowTypedMoRefWriteForm   string
	RelationshipRuleName       string
	MissingMoidMessage         string
	InvalidPayloadShapeMessage string
}

type DeleteDependencyRule struct {
	Endpoint    string
	RelationKey string
	Label       string
}
