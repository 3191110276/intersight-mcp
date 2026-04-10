package contracts

import (
	"strings"
	"testing"
)

func TestBuildSearchCatalogGroupsOperationsUnderResources(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"get":  {OperationID: "ListExampleWidgets"},
				"post": {OperationID: "CreateExampleWidget"},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Name": {Type: "string"},
				},
			},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.list": {
				SDKMethod: "example.widget.list",
				Tags:      []string{"example"},
				Resource:  "Error",
				Descriptor: OperationDescriptor{
					OperationID:  "ListExampleWidgets",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Widgets",
				},
				RelatedSchemas: []string{"example.Widget"},
			},
			"example.widget.create": {
				SDKMethod:           "example.widget.create",
				Summary:             "Create widget",
				Tags:                []string{"example"},
				Resource:            "example.Widget",
				RequestBodyRequired: true,
				RequestBodyFields:   []string{"Name"},
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
		},
	}
	rules := RuleCatalog{
		Metadata: meta,
		Methods: map[string]MethodRules{
			"example.widget.create": {
				SDKMethod:   "example.widget.create",
				OperationID: "CreateExampleWidget",
				Resource:    "example.Widget",
				Rules: []SemanticRule{
					{Kind: "conditional"},
				},
			},
		},
	}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	resource := search.Resources["example.widget"]
	if resource.Schema != "example.Widget" {
		t.Fatalf("schema = %q", resource.Schema)
	}
	if resource.CreateFields["Name"].Type != "string" {
		t.Fatalf("createFields[Name] = %#v, want string field metadata", resource.CreateFields["Name"])
	}
	if resource.Path != "/api/v1/example/Widgets" {
		t.Fatalf("resource path = %q", resource.Path)
	}
	if strings.Join(resource.Tags, ",") != "example" {
		t.Fatalf("tags = %#v", resource.Tags)
	}
	if len(resource.Rules) != 1 {
		t.Fatalf("resource rules = %#v", resource.Rules)
	}
	if strings.Join(search.ResourceNames, ",") != "example.widget" {
		t.Fatalf("resourceNames = %#v", search.ResourceNames)
	}

	if got := strings.Join(resource.Operations, ","); got != "create,list" {
		t.Fatalf("operations = %q, want create,list", got)
	}

	assertSearchPathIndex(t, search.Paths, "/api/v1/example/Widgets", "example.widget")
	assertSearchPathIndex(t, search.Paths, "/example/Widgets", "example.widget")
	assertSearchPathIndex(t, search.Paths, "/api/v1/example/widgets", "example.widget")
	assertSearchPathIndex(t, search.Paths, "/example/widgets", "example.widget")
}

func TestValidateSearchCatalogAgainstArtifactsRejectsMismatch(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets/{Moid}": {
				"get": {
					OperationID: "GetExampleWidget",
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Name": {Type: "string"},
				},
			},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.get": {
				SDKMethod: "example.widget.get",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "GetExampleWidget",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
		},
	}
	rules := RuleCatalog{Metadata: meta, Methods: map[string]MethodRules{}}
	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}
	resource := search.Resources["example.widget"]
	resource.Operations = []string{"list"}
	search.Resources["example.widget"] = resource

	err = ValidateSearchCatalogAgainstArtifacts(spec, catalog, rules, search)
	if err == nil || !strings.Contains(err.Error(), "does not match generated search catalog") {
		t.Fatalf("ValidateSearchCatalogAgainstArtifacts() error = %v, want mismatch failure", err)
	}
}

func TestBuildSearchCatalogIndexesSharedPathsAsSortedResourceKeys(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"get":  {OperationID: "ListExampleWidgets"},
				"post": {OperationID: "CreateExampleWidget"},
			},
			"/api/v1/example/Gadgets": {
				"get": {OperationID: "ListExampleGadgets"},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {Type: "object"},
			"example.Gadget": {Type: "object"},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.create": {
				SDKMethod: "example.widget.create",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
			"example.widget.list": {
				SDKMethod: "example.widget.list",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "ListExampleWidgets",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
			"example.gadget.list": {
				SDKMethod: "example.gadget.list",
				Resource:  "example.Gadget",
				Descriptor: OperationDescriptor{
					OperationID:  "ListExampleGadgets",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Gadgets",
				},
			},
		},
	}
	rules := RuleCatalog{Metadata: meta, Methods: map[string]MethodRules{}}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	got := search.Paths["/example/Widgets"]
	want := []string{"example.widget"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("paths[/example/Widgets] = %#v, want %#v", got, want)
	}
	if strings.Join(search.ResourceNames, ",") != "example.gadget,example.widget" {
		t.Fatalf("resourceNames = %#v", search.ResourceNames)
	}
}

func TestBuildSearchCatalogUsesCreateBodySubsetAndExcludesReadOnlyFields(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Required: true,
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{Circular: "example.Widget"},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Moid": {Type: "string", ReadOnly: true},
					"Name": {Type: "string"},
				},
			},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.create": {
				SDKMethod:           "example.widget.create",
				Resource:            "example.Widget",
				RequestBodyRequired: true,
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
		},
	}
	rules := RuleCatalog{Metadata: meta, Methods: map[string]MethodRules{}}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	resource := search.Resources["example.widget"]
	if _, exists := resource.CreateFields["Moid"]; exists {
		t.Fatalf("createFields = %#v, want readOnly fields removed", resource.CreateFields)
	}
	if resource.CreateFields["Name"].Type != "string" {
		t.Fatalf("createFields[Name] = %#v, want type string", resource.CreateFields["Name"])
	}
}

func TestValidateSearchMetricsCatalogAcceptsConsistentIndexes(t *testing.T) {
	t.Parallel()

	catalog := SearchMetricsCatalog{
		Groups: map[string]SearchMetricsGroup{
			"system.cpu": {
				Label:      "System CPU",
				DataSource: "PhysicalEntities",
				Dimensions: []string{"host.id", "host.name"},
				Metrics:    []string{"system.cpu.utilization_user"},
			},
		},
		ByName: map[string]SearchMetric{
			"system.cpu.utilization_user": {
				Name:       "system.cpu.utilization_user",
				Instrument: "system.cpu",
				DataSource: "PhysicalEntities",
			},
		},
		Examples: map[string]SearchMetricsExample{
			"cpu-breakdown": {
				MetricNames: []string{"system.cpu.utilization_user"},
			},
		},
	}

	if err := ValidateSearchMetricsCatalog(catalog); err != nil {
		t.Fatalf("ValidateSearchMetricsCatalog() error = %v", err)
	}
	normalized := NormalizeSearchMetricsCatalog(catalog)
	metric := normalized.ByName["system.cpu.utilization_user"]
	if got := strings.Join(metric.Dimensions, ","); got != "host.id,host.name" {
		t.Fatalf("metric.Dimensions = %#v, want inherited group dimensions", metric.Dimensions)
	}
}

func TestValidateSearchMetricsCatalogRejectsUnknownMetricReference(t *testing.T) {
	t.Parallel()

	catalog := SearchMetricsCatalog{
		Groups: map[string]SearchMetricsGroup{
			"system.cpu": {
				Label:   "System CPU",
				Metrics: []string{"system.cpu.utilization_user"},
			},
		},
	}

	err := ValidateSearchMetricsCatalog(catalog)
	if err == nil || !strings.Contains(err.Error(), "unknown metric") {
		t.Fatalf("ValidateSearchMetricsCatalog() error = %v, want unknown metric failure", err)
	}
}

func TestBuildSearchCatalogMergesCreateRulesIntoFieldMetadata(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Required: true,
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{Circular: "example.Widget"},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {
				Type: "object",
				Required: []string{
					"Name",
				},
				Properties: map[string]*NormalizedSchema{
					"Name":         {Type: "string"},
					"Mode":         {Type: "string"},
					"Organization": {Circular: "organization.Organization.Relationship"},
				},
			},
			"organization.Organization.Relationship": {
				Type:                   "object",
				Relationship:           true,
				RelationshipTarget:     "organization.Organization",
				RelationshipWriteForms: []string{"moidRef", "typedMoRef"},
				Properties: map[string]*NormalizedSchema{
					"Moid": {Type: "string"},
				},
			},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.create": {
				SDKMethod:           "example.widget.create",
				Resource:            "example.Widget",
				RequestBodyRequired: true,
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
		},
	}
	rules := RuleCatalog{
		Metadata: meta,
		Methods: map[string]MethodRules{
			"example.widget.create": {
				SDKMethod:   "example.widget.create",
				OperationID: "CreateExampleWidget",
				Resource:    "example.Widget",
				Rules: []SemanticRule{
					{Kind: "one_of", RequireAny: []FieldRule{{Field: "Mode"}, {Field: "Organization", Target: "organization.Organization"}}},
					{Kind: "conditional", When: &RuleCondition{Field: "Mode", Equals: "fast"}, Require: []FieldRule{{Field: "Organization"}}},
				},
			},
		},
	}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	resource := search.Resources["example.widget"]
	if !resource.CreateFields["Name"].Required {
		t.Fatalf("createFields[Name] = %#v, want required=true from schema", resource.CreateFields["Name"])
	}
	if got := strings.Join(resource.CreateFields["Mode"].OneOf, ","); got != "Mode,Organization" {
		t.Fatalf("createFields[Mode].OneOf = %#v, want Mode,Organization", resource.CreateFields["Mode"].OneOf)
	}
	if got := resource.CreateFields["Organization"].Example; got == nil {
		t.Fatalf("createFields[Organization].Example = %#v, want relationship example", got)
	}
	if len(resource.Rules) != 1 || resource.Rules[0].Kind != "conditional" {
		t.Fatalf("resource rules = %#v, want only conditional rule retained", resource.Rules)
	}
}

func TestBuildSearchCatalogDedupesHoistedRulesAcrossWriteAliases(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets/{Moid}": {
				"patch": {OperationID: "PatchExampleWidget"},
				"post":  {OperationID: "PostExampleWidget"},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {Type: "object"},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.post": {
				SDKMethod: "example.widget.post",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "PostExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
			"example.widget.update": {
				SDKMethod: "example.widget.update",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "PatchExampleWidget",
					Method:       "PATCH",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
		},
	}
	sharedRules := []SemanticRule{
		{Kind: "conditional", When: &RuleCondition{Field: "Mode", Equals: "fast"}, Require: []FieldRule{{Field: "Name"}}},
	}
	rules := RuleCatalog{
		Metadata: meta,
		Methods: map[string]MethodRules{
			"example.widget.post": {
				SDKMethod:   "example.widget.post",
				OperationID: "PostExampleWidget",
				Resource:    "example.Widget",
				Rules:       append([]SemanticRule(nil), sharedRules...),
			},
			"example.widget.update": {
				SDKMethod:   "example.widget.update",
				OperationID: "PatchExampleWidget",
				Resource:    "example.Widget",
				Rules:       append([]SemanticRule(nil), sharedRules...),
			},
		},
	}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	resource := search.Resources["example.widget"]
	if len(resource.Rules) != 1 || resource.Rules[0].Kind != "conditional" {
		t.Fatalf("resource rules = %#v, want conditional rule retained once at resource level", resource.Rules)
	}
	if len(resource.CreateFields) != 0 {
		t.Fatalf("createFields = %#v, want none without a create operation", resource.CreateFields)
	}
}

func TestBuildSearchCatalogHoistsCollectionAndItemPathsToResourceLevel(t *testing.T) {
	t.Parallel()

	meta := ArtifactSourceMetadata{
		PublishedVersion: "1.0.0-test",
		SourceURL:        "https://example.com/spec",
		SHA256:           "abc123",
		RetrievalDate:    "2026-04-08",
	}
	spec := NormalizedSpec{
		Metadata: meta,
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"get":  {OperationID: "ListExampleWidgets"},
				"post": {OperationID: "CreateExampleWidget"},
			},
			"/api/v1/example/Widgets/{Moid}": {
				"get":    {OperationID: "GetExampleWidget"},
				"patch":  {OperationID: "UpdateExampleWidget"},
				"delete": {OperationID: "DeleteExampleWidget"},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget": {Type: "object"},
		},
	}
	catalog := SDKCatalog{
		Metadata: meta,
		Methods: map[string]SDKMethod{
			"example.widget.list": {
				SDKMethod: "example.widget.list",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "ListExampleWidgets",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
			"example.widget.create": {
				SDKMethod: "example.widget.create",
				Resource:  "example.Widget",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
			"example.widget.get": {
				SDKMethod:      "example.widget.get",
				Resource:       "example.Widget",
				PathParameters: []string{"Moid"},
				Descriptor: OperationDescriptor{
					OperationID:  "GetExampleWidget",
					Method:       "GET",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
			"example.widget.update": {
				SDKMethod:      "example.widget.update",
				Resource:       "example.Widget",
				PathParameters: []string{"Moid"},
				Descriptor: OperationDescriptor{
					OperationID:  "UpdateExampleWidget",
					Method:       "PATCH",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
			"example.widget.delete": {
				SDKMethod:      "example.widget.delete",
				Resource:       "example.Widget",
				PathParameters: []string{"Moid"},
				Descriptor: OperationDescriptor{
					OperationID:  "DeleteExampleWidget",
					Method:       "DELETE",
					PathTemplate: "/api/v1/example/Widgets/{Moid}",
				},
			},
		},
	}
	rules := RuleCatalog{Metadata: meta, Methods: map[string]MethodRules{}}

	search, err := BuildSearchCatalog(spec, catalog, rules, SearchMetricsCatalog{})
	if err != nil {
		t.Fatalf("BuildSearchCatalog() error = %v", err)
	}

	resource := search.Resources["example.widget"]
	if resource.Path != "/api/v1/example/Widgets/{Moid?}" {
		t.Fatalf("resource path = %q", resource.Path)
	}
	if got := strings.Join(resource.Operations, ","); got != "create,delete,get,list,update" {
		t.Fatalf("operations = %q, want create,delete,get,list,update", got)
	}

	assertSearchPathIndex(t, search.Paths, "/api/v1/example/Widgets", "example.widget")
	assertSearchPathIndex(t, search.Paths, "/api/v1/example/Widgets/{Moid}", "example.widget")
}

func assertSearchPathIndex(t *testing.T, index map[string][]string, key string, want ...string) {
	t.Helper()

	got := index[key]
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("paths[%q] = %#v, want %#v", key, got, want)
	}
}
