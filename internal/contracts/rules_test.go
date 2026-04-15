package contracts

import (
	"reflect"
	"strings"
	"testing"
)

func TestSchemaAtFieldPathSupportsArraySegments(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Schemas: map[string]NormalizedSchema{
			"example.Child": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Leaf": {Type: "string"},
				},
			},
		},
	}
	root := &NormalizedSchema{
		Type: "object",
		Properties: map[string]*NormalizedSchema{
			"Items": {
				Type:  "array",
				Items: &NormalizedSchema{Circular: "example.Child"},
			},
		},
	}

	schema, ok := schemaAtFieldPath(spec, root, "Items[].Leaf")
	if !ok {
		t.Fatalf("schemaAtFieldPath returned !ok")
	}
	if schema.Type != "string" {
		t.Fatalf("schema.Type = %q, want string", schema.Type)
	}
}

func testIntersightRuleTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			SDKMethod: "vnic.ethIf.create",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				NewRequiredRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				NewRequiredRule("EthAdapterPolicy", "vnic.EthAdapterPolicy"),
				NewRequiredRule("EthQosPolicy", "vnic.EthQosPolicy"),
				NewRequiredRule("FabricEthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				NewRequiredRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.post",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				NewRequiredRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				NewRequiredRule("EthAdapterPolicy", "vnic.EthAdapterPolicy"),
				NewRequiredRule("EthQosPolicy", "vnic.EthQosPolicy"),
				NewRequiredRule("FabricEthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				NewRequiredRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "aaa.retentionPolicy.create",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("RetentionPeriod", ""),
				NewMinimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "access.policy.create",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				NewRequiredRule("AddressType", ""),
				NewRequiredRule("ConfigurationType", ""),
				NewConditionalRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				NewConditionalMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.create",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				NewForbidRule("Name"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.create",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				NewRequiredRule("StartDate", ""),
				NewOneOfRule("Entity", "AlarmRules"),
			},
		},
		{
			SDKMethod: "cond.thresholdDefinition.create",
			Resource:  "cond.ThresholdDefinition",
			Rules: []SemanticRule{
				NewRequiredRule("Condition", ""),
			},
		},
		{
			SDKMethod: "deviceconnector.policy.create",
			Resource:  "deviceconnector.Policy",
			Rules: []SemanticRule{
				NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.create",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				NewRequiredRule("ExtaTraffic", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.create",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				NewRequiredRule("ExtaTraffic", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.create",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("HxdpRootPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.create",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.create",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("Hostname", ""),
				NewMinimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.create",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("HxdpVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.create",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("DnsServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.create",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("DataCenter", ""),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.create",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("Enabled", ""),
			},
		},
		{
			SDKMethod: "ntp.policy.create",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				NewRequiredRule("Name", ""),
				NewRequiredRule("Enabled", ""),
				NewRequiredRule("Timezone", ""),
				NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "organization.organization.create",
			Resource:  "organization.Organization",
			Rules: []SemanticRule{
				NewRequiredRule("Name", ""),
			},
		},
		{
			SDKMethod: "fabric.portPolicy.create",
			Resource:  "fabric.PortPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("Name", ""),
				NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "server.profile.create",
			Resource:  "server.Profile",
			Rules: []SemanticRule{
				NewRequiredRule("Name", ""),
				NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.create",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("Schedule", ""),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.create",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				NewRequiredRule("ScheduleParams", "", 1),
				NewRequiredRule("ScheduleParams[].Name", ""),
			},
		},
		{
			SDKMethod: "recovery.backupProfile.create",
			Resource:  "recovery.BackupProfile",
			Rules: []SemanticRule{
				NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "resourcepool.pool.create",
			Resource:  "resourcepool.Pool",
			Rules: []SemanticRule{
				NewConditionalRequireRule("ResourceType", "Server", FieldRule{Field: "ResourcePoolParameters"}),
			},
		},
		{
			SDKMethod: "smtp.policy.create",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				NewRequiredRule("Enabled", ""),
			},
		},
		{
			SDKMethod: "smtp.policyTest.create",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				NewRequiredRule("Policy", "smtp.Policy"),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.create",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "server.diagnostics.create",
			Resource:  "server.Diagnostics",
			Rules: []SemanticRule{
				NewRequiredRule("Server", "compute.Physical"),
				NewRequiredRule("ComponentList", "", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.create",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				NewRequiredRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.create",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				NewMinimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.fcAdapterPolicies.create",
			Resource:  "vnic.FcAdapterPolicy",
			Rules: []SemanticRule{
				NewMinimumRule(MinimumRule{Field: "ErrorDetectionTimeout", Value: 1000}),
			},
		},
		{
			SDKMethod: "vnic.fcQosPolicies.create",
			Resource:  "vnic.FcQosPolicy",
			Rules: []SemanticRule{
				NewMinimumRule(MinimumRule{Field: "MaxDataFieldSize", Value: 256}),
			},
		},
		{
			SDKMethod: "workflow.ansibleBatchExecutors.create",
			Resource:  "workflow.AnsibleBatchExecutor",
			Rules: []SemanticRule{
				NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.powerShellBatchApiExecutors.create",
			Resource:  "workflow.PowerShellBatchApiExecutor",
			Rules: []SemanticRule{
				NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.sshBatchExecutors.create",
			Resource:  "workflow.SshBatchExecutor",
			Rules: []SemanticRule{
				NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.workflowInfos.create",
			Resource:  "workflow.WorkflowInfo",
			Rules: []SemanticRule{
				NewMinimumRule(MinimumRule{Field: "FailedWorkflowCleanupDuration", Value: 1}),
				NewMinimumRule(MinimumRule{Field: "SuccessWorkflowCleanupDuration", Value: 1}),
				NewConditionalForbidRule("Action", "None", "Action"),
			},
		},
		{
			SDKMethod: "workload.blueprints.create",
			Resource:  "workload.Blueprint",
			Rules: []SemanticRule{
				NewRequiredRule("Label", ""),
				NewRequiredRule("ServiceItems", "", 1),
				NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9_]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDefinitions.create",
			Resource:  "workload.WorkloadDefinition",
			Rules: []SemanticRule{
				NewRequiredRule("Blueprints", "", 1),
				NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.create",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				NewRequiredRule("IpAddress", ""),
			},
		},
	}
}

func TestBuildRuleCatalogIncludesPostWriteMethodsForPhaseFourResources(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/ntp/Policies": {
				"post": {
					OperationID: "CreateNtpPolicy",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Name":                    {Type: "string"},
										"Enabled":                 {Type: "boolean"},
										"Timezone":                {Type: "string"},
										"NtpServers":              {Type: "array", Items: &NormalizedSchema{Type: "string"}},
										"AuthenticatedNtpServers": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/organization/Organizations": {
				"post": {
					OperationID: "CreateOrganizationOrganization",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Name": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/fabric/PortPolicies": {
				"post": {
					OperationID: "CreateFabricPortPolicy",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Name":         {Type: "string"},
										"Organization": {Relationship: true, RelationshipTarget: "organization.Organization"},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/server/Profiles": {
				"post": {
					OperationID: "CreateServerProfile",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Name":         {Type: "string"},
										"Organization": {Relationship: true, RelationshipTarget: "organization.Organization"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"organization.Organization": {Type: "object"},
			"fabric.PortPolicy":         {Type: "object"},
			"server.Profile":            {Type: "object"},
			"ntp.Policy":                {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"vnic.ethIf.create": {
				SDKMethod: "vnic.ethIf.create",
				Descriptor: OperationDescriptor{
					OperationID: "CreateVnicEthIf",
					Method:      "POST",
				},
			},
			"vnic.ethIf.post": {
				SDKMethod: "vnic.ethIf.post",
				Descriptor: OperationDescriptor{
					OperationID: "UpdateVnicEthIf",
					Method:      "POST",
				},
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	createRules, ok := rules.Methods["vnic.ethIf.create"]
	if !ok {
		t.Fatalf("expected rules for vnic.ethIf.create")
	}
	postRules, ok := rules.Methods["vnic.ethIf.post"]
	if !ok {
		t.Fatalf("expected rules for vnic.ethIf.post")
	}
	if createRules.Resource != "vnic.EthIf" || postRules.Resource != "vnic.EthIf" {
		t.Fatalf("unexpected resources: create=%q post=%q", createRules.Resource, postRules.Resource)
	}
	if !reflect.DeepEqual(createRules.Rules, postRules.Rules) {
		t.Fatalf("expected post rules to match create rules")
	}
}

func TestBuildRuleCatalogOmitsRequiredRulesForEthIf(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/vnic/EthIfs": {
				"post": {
					OperationID: "CreateVnicEthIf",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Required: []string{
										"LanConnectivityPolicy",
										"EthAdapterPolicy",
										"EthQosPolicy",
										"FabricEthNetworkControlPolicy",
										"FabricEthNetworkGroupPolicy",
									},
									Properties: map[string]*NormalizedSchema{
										"LanConnectivityPolicy":         {Relationship: true, RelationshipTarget: "vnic.LanConnectivityPolicy"},
										"EthAdapterPolicy":              {Relationship: true, RelationshipTarget: "vnic.EthAdapterPolicy"},
										"EthQosPolicy":                  {Relationship: true, RelationshipTarget: "vnic.EthQosPolicy"},
										"FabricEthNetworkControlPolicy": {Relationship: true, RelationshipTarget: "fabric.EthNetworkControlPolicy"},
										"FabricEthNetworkGroupPolicy": {
											Type:  "array",
											Items: &NormalizedSchema{Relationship: true, RelationshipTarget: "fabric.EthNetworkGroupPolicy"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"vnic.ethIf.create": {
				SDKMethod: "vnic.ethIf.create",
				Descriptor: OperationDescriptor{
					OperationID: "CreateVnicEthIf",
					Method:      "POST",
				},
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	createRules, ok := rules.Methods["vnic.ethIf.create"]
	if !ok {
		t.Fatalf("expected rules for vnic.ethIf.create")
	}

	for _, rule := range createRules.Rules {
		if rule.Kind == "required" {
			t.Fatalf("unexpected required rule retained: %#v", rule)
		}
	}
}

func TestBuildRuleCatalogPreservesBackendRequiredRulesWhenSchemaLeavesFieldsOptional(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/vnic/EthIfs": {
				"post": {
					OperationID: "CreateVnicEthIf",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Required: []string{
										"LanConnectivityPolicy",
									},
									Properties: map[string]*NormalizedSchema{
										"LanConnectivityPolicy":         {Relationship: true, RelationshipTarget: "vnic.LanConnectivityPolicy"},
										"EthAdapterPolicy":              {Relationship: true, RelationshipTarget: "vnic.EthAdapterPolicy"},
										"EthQosPolicy":                  {Relationship: true, RelationshipTarget: "vnic.EthQosPolicy"},
										"FabricEthNetworkControlPolicy": {Relationship: true, RelationshipTarget: "fabric.EthNetworkControlPolicy"},
										"FabricEthNetworkGroupPolicy": {
											Type:  "array",
											Items: &NormalizedSchema{Relationship: true, RelationshipTarget: "fabric.EthNetworkGroupPolicy"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"vnic.ethIf.create": {
				SDKMethod: "vnic.ethIf.create",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateVnicEthIf",
					Method:       "POST",
					PathTemplate: "/api/v1/vnic/EthIfs",
				},
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	createRules, ok := rules.Methods["vnic.ethIf.create"]
	if !ok {
		t.Fatalf("expected rules for vnic.ethIf.create")
	}

	var requiredFields []string
	for _, rule := range createRules.Rules {
		if rule.Kind != "required" || len(rule.Require) == 0 {
			continue
		}
		requiredFields = append(requiredFields, rule.Require[0].Field)
	}

	want := []string{
		"EthAdapterPolicy",
		"EthQosPolicy",
		"FabricEthNetworkControlPolicy",
		"FabricEthNetworkGroupPolicy",
	}
	if !reflect.DeepEqual(requiredFields, want) {
		t.Fatalf("required fields = %#v, want %#v", requiredFields, want)
	}
}

func TestBuildRuleCatalogPreservesAlarmSuppressionStartDateRequirementWhenSchemaLeavesItOptional(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/cond/AlarmSuppressions": {
				"post": {
					OperationID: "CreateCondAlarmSuppression",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"StartDate":  {Type: "string", Format: "date-time"},
										"Entity":     {Relationship: true, RelationshipTarget: "mo.BaseMo"},
										"AlarmRules": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"cond.alarmSuppression.create": {
				SDKMethod: "cond.alarmSuppression.create",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateCondAlarmSuppression",
					Method:       "POST",
					PathTemplate: "/api/v1/cond/AlarmSuppressions",
				},
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	createRules, ok := rules.Methods["cond.alarmSuppression.create"]
	if !ok {
		t.Fatalf("expected rules for cond.alarmSuppression.create")
	}
	if len(createRules.Rules) != 2 {
		t.Fatalf("cond.alarmSuppression.create rules = %#v, want required+one_of", createRules.Rules)
	}
	if len(createRules.Rules[0].Require) != 1 || createRules.Rules[0].Require[0].Field != "StartDate" {
		t.Fatalf("unexpected alarm suppression required rule: %#v", createRules.Rules[0])
	}
	if createRules.Rules[1].Kind != "one_of" || len(createRules.Rules[1].RequireAny) != 2 {
		t.Fatalf("unexpected alarm suppression one-of rule: %#v", createRules.Rules[1])
	}
	if createRules.Rules[1].RequireAny[0].Field != "Entity" || createRules.Rules[1].RequireAny[1].Field != "AlarmRules" {
		t.Fatalf("unexpected alarm suppression one-of fields: %#v", createRules.Rules[1].RequireAny)
	}
}

func TestBuildRuleCatalogIncludesPolicyCreateRulesFromProbeFindings(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/workflow/AnsibleBatchExecutors": {
				"post": {
					OperationID: "CreateWorkflowAnsibleBatchExecutor",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Batch": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/workflow/PowerShellBatchApiExecutors": {
				"post": {
					OperationID: "CreateWorkflowPowerShellBatchApiExecutor",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Batch": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/workflow/SshBatchExecutors": {
				"post": {
					OperationID: "CreateWorkflowSshBatchExecutor",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Batch": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/workload/Blueprints": {
				"post": {
					OperationID: "CreateWorkloadBlueprint",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Label":        {Type: "string"},
										"Name":         {Type: "string"},
										"ServiceItems": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/workload/WorkloadDefinitions": {
				"post": {
					OperationID: "CreateWorkloadWorkloadDefinition",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Blueprints": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
										"Name":       {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"workflow.AnsibleBatchExecutor":       {Type: "object"},
			"workflow.PowerShellBatchApiExecutor": {Type: "object"},
			"workflow.SshBatchExecutor":           {Type: "object"},
			"workflow.WorkflowInfo":               {Type: "object"},
			"workload.Blueprint":                  {Type: "object"},
			"workload.WorkloadDefinition":         {Type: "object"},
			"vnic.FcAdapterPolicy":                {Type: "object"},
			"vnic.FcQosPolicy":                    {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"aaa.retentionPolicy.create":                  {SDKMethod: "aaa.retentionPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateAaaRetentionPolicy", Method: "POST"}},
			"access.policy.create":                        {SDKMethod: "access.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateAccessPolicy", Method: "POST"}},
			"appliance.dataExportPolicy.create":           {SDKMethod: "appliance.dataExportPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateApplianceDataExportPolicy", Method: "POST"}},
			"cond.alarmSuppression.create":                {SDKMethod: "cond.alarmSuppression.create", Descriptor: OperationDescriptor{OperationID: "CreateCondAlarmSuppression", Method: "POST"}},
			"hyperflex.extFcStoragePolicy.create":         {SDKMethod: "hyperflex.extFcStoragePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexExtFcStoragePolicy", Method: "POST"}},
			"hyperflex.extIscsiStoragePolicy.create":      {SDKMethod: "hyperflex.extIscsiStoragePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexExtIscsiStoragePolicy", Method: "POST"}},
			"hyperflex.localCredentialPolicy.create":      {SDKMethod: "hyperflex.localCredentialPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexLocalCredentialPolicy", Method: "POST"}},
			"hyperflex.nodeConfigPolicy.create":           {SDKMethod: "hyperflex.nodeConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexNodeConfigPolicy", Method: "POST"}},
			"hyperflex.proxySettingPolicy.create":         {SDKMethod: "hyperflex.proxySettingPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexProxySettingPolicy", Method: "POST"}},
			"hyperflex.softwareVersionPolicy.create":      {SDKMethod: "hyperflex.softwareVersionPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexSoftwareVersionPolicy", Method: "POST"}},
			"hyperflex.sysConfigPolicy.create":            {SDKMethod: "hyperflex.sysConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexSysConfigPolicy", Method: "POST"}},
			"hyperflex.vcenterConfigPolicy.create":        {SDKMethod: "hyperflex.vcenterConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexVcenterConfigPolicy", Method: "POST"}},
			"iam.ldapPolicy.create":                       {SDKMethod: "iam.ldapPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateIamLdapPolicy", Method: "POST"}},
			"ntp.policy.create":                           {SDKMethod: "ntp.policy.create", Resource: "ntp.Policy", Descriptor: OperationDescriptor{OperationID: "CreateNtpPolicy", Method: "POST", PathTemplate: "/api/v1/ntp/Policies"}},
			"organization.organization.create":            {SDKMethod: "organization.organization.create", Resource: "organization.Organization", Descriptor: OperationDescriptor{OperationID: "CreateOrganizationOrganization", Method: "POST", PathTemplate: "/api/v1/organization/Organizations"}},
			"fabric.portPolicy.create":                    {SDKMethod: "fabric.portPolicy.create", Resource: "fabric.PortPolicy", Descriptor: OperationDescriptor{OperationID: "CreateFabricPortPolicy", Method: "POST", PathTemplate: "/api/v1/fabric/PortPolicies"}},
			"recovery.scheduleConfigPolicy.create":        {SDKMethod: "recovery.scheduleConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateRecoveryScheduleConfigPolicy", Method: "POST"}},
			"scheduler.schedulePolicy.create":             {SDKMethod: "scheduler.schedulePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateSchedulerSchedulePolicy", Method: "POST"}},
			"server.profile.create":                       {SDKMethod: "server.profile.create", Resource: "server.Profile", Descriptor: OperationDescriptor{OperationID: "CreateServerProfile", Method: "POST", PathTemplate: "/api/v1/server/Profiles"}},
			"smtp.policy.create":                          {SDKMethod: "smtp.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateSmtpPolicy", Method: "POST"}},
			"smtp.policyTest.create":                      {SDKMethod: "smtp.policyTest.create", Descriptor: OperationDescriptor{OperationID: "CreateSmtpPolicyTest", Method: "POST"}},
			"storage.driveSecurityPolicy.create":          {SDKMethod: "storage.driveSecurityPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateStorageDriveSecurityPolicy", Method: "POST"}},
			"syslog.policy.create":                        {SDKMethod: "syslog.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateSyslogPolicy", Method: "POST"}},
			"vnic.fcAdapterPolicies.create":               {SDKMethod: "vnic.fcAdapterPolicies.create", Resource: "vnic.FcAdapterPolicy", Descriptor: OperationDescriptor{OperationID: "CreateVnicFcAdapterPolicy", Method: "POST", PathTemplate: "/api/v1/vnic/FcAdapterPolicies"}},
			"vnic.fcQosPolicies.create":                   {SDKMethod: "vnic.fcQosPolicies.create", Resource: "vnic.FcQosPolicy", Descriptor: OperationDescriptor{OperationID: "CreateVnicFcQosPolicy", Method: "POST", PathTemplate: "/api/v1/vnic/FcQosPolicies"}},
			"vnic.iscsiAdapterPolicy.create":              {SDKMethod: "vnic.iscsiAdapterPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateVnicIscsiAdapterPolicy", Method: "POST"}},
			"vnic.iscsiStaticTargetPolicy.create":         {SDKMethod: "vnic.iscsiStaticTargetPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateVnicIscsiStaticTargetPolicy", Method: "POST"}},
			"workflow.ansibleBatchExecutors.create":       {SDKMethod: "workflow.ansibleBatchExecutors.create", Resource: "workflow.AnsibleBatchExecutor", Descriptor: OperationDescriptor{OperationID: "CreateWorkflowAnsibleBatchExecutor", Method: "POST", PathTemplate: "/api/v1/workflow/AnsibleBatchExecutors"}},
			"workflow.powerShellBatchApiExecutors.create": {SDKMethod: "workflow.powerShellBatchApiExecutors.create", Resource: "workflow.PowerShellBatchApiExecutor", Descriptor: OperationDescriptor{OperationID: "CreateWorkflowPowerShellBatchApiExecutor", Method: "POST", PathTemplate: "/api/v1/workflow/PowerShellBatchApiExecutors"}},
			"workflow.sshBatchExecutors.create":           {SDKMethod: "workflow.sshBatchExecutors.create", Resource: "workflow.SshBatchExecutor", Descriptor: OperationDescriptor{OperationID: "CreateWorkflowSshBatchExecutor", Method: "POST", PathTemplate: "/api/v1/workflow/SshBatchExecutors"}},
			"workflow.workflowInfos.create":               {SDKMethod: "workflow.workflowInfos.create", Resource: "workflow.WorkflowInfo", Descriptor: OperationDescriptor{OperationID: "CreateWorkflowWorkflowInfo", Method: "POST", PathTemplate: "/api/v1/workflow/WorkflowInfos"}},
			"workload.blueprints.create":                  {SDKMethod: "workload.blueprints.create", Resource: "workload.Blueprint", Descriptor: OperationDescriptor{OperationID: "CreateWorkloadBlueprint", Method: "POST", PathTemplate: "/api/v1/workload/Blueprints"}},
			"workload.workloadDefinitions.create":         {SDKMethod: "workload.workloadDefinitions.create", Resource: "workload.WorkloadDefinition", Descriptor: OperationDescriptor{OperationID: "CreateWorkloadWorkloadDefinition", Method: "POST", PathTemplate: "/api/v1/workload/WorkloadDefinitions"}},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	retention := rules.Methods["aaa.retentionPolicy.create"]
	if len(retention.Rules) != 1 {
		t.Fatalf("aaa.retentionPolicy.create rules = %#v, want minimum only", retention.Rules)
	}
	if retention.Rules[0].Minimum[0].Field != "RetentionPeriod" || retention.Rules[0].Minimum[0].Value != 6 {
		t.Fatalf("unexpected retention minimum: %#v", retention.Rules[0])
	}

	dataExport := rules.Methods["appliance.dataExportPolicy.create"]
	if len(dataExport.Rules) != 1 || !reflect.DeepEqual(dataExport.Rules[0].Forbid, []string{"Name"}) {
		t.Fatalf("unexpected data export rules: %#v", dataExport.Rules)
	}

	access := rules.Methods["access.policy.create"]
	if len(access.Rules) != 2 {
		t.Fatalf("access.policy.create rules = %#v, want inband conditional+minimum only", access.Rules)
	}
	if access.Rules[0].Require[0].Field != "InbandIpPool" || access.Rules[0].Require[0].Target != "ippool.Pool" {
		t.Fatalf("unexpected access inband pool requirement: %#v", access.Rules[0])
	}
	if access.Rules[1].Minimum[0].Field != "InbandVlan" || access.Rules[1].Minimum[0].Value != 4 {
		t.Fatalf("unexpected access inband vlan minimum: %#v", access.Rules[1])
	}

	ntp := rules.Methods["ntp.policy.create"]
	if len(ntp.Rules) != 1 {
		t.Fatalf("ntp.policy.create rules = %#v, want one_of only", ntp.Rules)
	}
	if ntp.Rules[0].Kind != "one_of" || len(ntp.Rules[0].RequireAny) != 2 {
		t.Fatalf("unexpected ntp one-of rule: %#v", ntp.Rules[0])
	}
	if ntp.Rules[0].RequireAny[0].Field != "NtpServers" || ntp.Rules[0].RequireAny[1].Field != "AuthenticatedNtpServers" {
		t.Fatalf("unexpected ntp one-of fields: %#v", ntp.Rules[0].RequireAny)
	}

	alarmSuppression := rules.Methods["cond.alarmSuppression.create"]
	if len(alarmSuppression.Rules) != 1 {
		t.Fatalf("cond.alarmSuppression.create rules = %#v, want one_of only", alarmSuppression.Rules)
	}
	if alarmSuppression.Rules[0].Kind != "one_of" || len(alarmSuppression.Rules[0].RequireAny) != 2 {
		t.Fatalf("unexpected alarm suppression one-of rule: %#v", alarmSuppression.Rules[0])
	}
	if alarmSuppression.Rules[0].RequireAny[0].Field != "Entity" || alarmSuppression.Rules[0].RequireAny[1].Field != "AlarmRules" {
		t.Fatalf("unexpected alarm suppression one-of fields: %#v", alarmSuppression.Rules[0].RequireAny)
	}

	smtp := rules.Methods["smtp.policy.create"]
	if len(smtp.Rules) != 0 {
		t.Fatalf("smtp.policy.create rules = %#v, want no custom rules", smtp.Rules)
	}

	smtpTest := rules.Methods["smtp.policyTest.create"]
	if len(smtpTest.Rules) != 0 {
		t.Fatalf("unexpected smtp policy test rules: %#v", smtpTest.Rules)
	}

	syslog := rules.Methods["syslog.policy.create"]
	if len(syslog.Rules) != 0 {
		t.Fatalf("unexpected syslog rules: %#v", syslog.Rules)
	}

	iscsiAdapter := rules.Methods["vnic.iscsiAdapterPolicy.create"]
	if len(iscsiAdapter.Rules) != 1 || len(iscsiAdapter.Rules[0].Minimum) != 1 {
		t.Fatalf("unexpected iSCSI adapter rules: %#v", iscsiAdapter.Rules)
	}
	if iscsiAdapter.Rules[0].Minimum[0].Field != "DhcpTimeout" || iscsiAdapter.Rules[0].Minimum[0].Value != 60 {
		t.Fatalf("unexpected iSCSI adapter minimum: %#v", iscsiAdapter.Rules[0])
	}

	fcAdapter := rules.Methods["vnic.fcAdapterPolicies.create"]
	if len(fcAdapter.Rules) != 1 || len(fcAdapter.Rules[0].Minimum) != 1 {
		t.Fatalf("unexpected fc adapter rules: %#v", fcAdapter.Rules)
	}
	if fcAdapter.Rules[0].Minimum[0].Field != "ErrorDetectionTimeout" || fcAdapter.Rules[0].Minimum[0].Value != 1000 {
		t.Fatalf("unexpected fc adapter minimum: %#v", fcAdapter.Rules[0])
	}

	fcQos := rules.Methods["vnic.fcQosPolicies.create"]
	if len(fcQos.Rules) != 1 || len(fcQos.Rules[0].Minimum) != 1 {
		t.Fatalf("unexpected fc qos rules: %#v", fcQos.Rules)
	}
	if fcQos.Rules[0].Minimum[0].Field != "MaxDataFieldSize" || fcQos.Rules[0].Minimum[0].Value != 256 {
		t.Fatalf("unexpected fc qos minimum: %#v", fcQos.Rules[0])
	}

	for _, sdkMethod := range []string{
		"workflow.ansibleBatchExecutors.create",
		"workflow.powerShellBatchApiExecutors.create",
		"workflow.sshBatchExecutors.create",
	} {
		got := rules.Methods[sdkMethod].Rules
		if len(got) != 1 || len(got[0].Require) != 1 || got[0].Require[0].Field != "Batch" || got[0].Require[0].MinCount != 1 {
			t.Fatalf("unexpected batch executor rules for %s: %#v", sdkMethod, got)
		}
	}

	workflowInfo := rules.Methods["workflow.workflowInfos.create"]
	if len(workflowInfo.Rules) != 3 {
		t.Fatalf("unexpected workflow info rules: %#v", workflowInfo.Rules)
	}
	if workflowInfo.Rules[0].Minimum[0].Field != "FailedWorkflowCleanupDuration" || workflowInfo.Rules[0].Minimum[0].Value != 1 {
		t.Fatalf("unexpected workflow info failed cleanup minimum: %#v", workflowInfo.Rules[0])
	}
	if workflowInfo.Rules[1].Minimum[0].Field != "SuccessWorkflowCleanupDuration" || workflowInfo.Rules[1].Minimum[0].Value != 1 {
		t.Fatalf("unexpected workflow info success cleanup minimum: %#v", workflowInfo.Rules[1])
	}
	if workflowInfo.Rules[2].When == nil || workflowInfo.Rules[2].When.Field != "Action" || workflowInfo.Rules[2].When.Equals != "None" || !reflect.DeepEqual(workflowInfo.Rules[2].Forbid, []string{"Action"}) {
		t.Fatalf("unexpected workflow info action rule: %#v", workflowInfo.Rules[2])
	}

	blueprint := rules.Methods["workload.blueprints.create"]
	if len(blueprint.Rules) != 3 {
		t.Fatalf("unexpected blueprint rules: %#v", blueprint.Rules)
	}
	if blueprint.Rules[0].Kind != "required" || blueprint.Rules[0].Require[0].Field != "Label" {
		t.Fatalf("unexpected blueprint label rule: %#v", blueprint.Rules[0])
	}
	if blueprint.Rules[1].Kind != "required" || blueprint.Rules[1].Require[0].Field != "ServiceItems" || blueprint.Rules[1].Require[0].MinCount != 1 {
		t.Fatalf("unexpected blueprint service item rule: %#v", blueprint.Rules[1])
	}
	if blueprint.Rules[2].Kind != "pattern" || blueprint.Rules[2].Pattern[0].Field != "Name" || blueprint.Rules[2].Pattern[0].Value != "^[a-zA-Z0-9][a-zA-Z0-9_]{0,31}$" {
		t.Fatalf("unexpected blueprint name rule: %#v", blueprint.Rules[2])
	}

	workloadDefinition := rules.Methods["workload.workloadDefinitions.create"]
	if len(workloadDefinition.Rules) != 2 {
		t.Fatalf("unexpected workload definition rules: %#v", workloadDefinition.Rules)
	}
	if workloadDefinition.Rules[0].Kind != "required" || workloadDefinition.Rules[0].Require[0].Field != "Blueprints" || workloadDefinition.Rules[0].Require[0].MinCount != 1 {
		t.Fatalf("unexpected workload definition blueprint rule: %#v", workloadDefinition.Rules[0])
	}
	if workloadDefinition.Rules[1].Kind != "pattern" || workloadDefinition.Rules[1].Pattern[0].Field != "Name" {
		t.Fatalf("unexpected workload definition name rule: %#v", workloadDefinition.Rules[1])
	}

	localCreds := rules.Methods["hyperflex.localCredentialPolicy.create"]
	if len(localCreds.Rules) != 0 {
		t.Fatalf("hyperflex.localCredentialPolicy.create rules = %#v, want no custom rules", localCreds.Rules)
	}

	nodeConfig := rules.Methods["hyperflex.nodeConfigPolicy.create"]
	if len(nodeConfig.Rules) != 0 {
		t.Fatalf("unexpected hyperflex node config rules: %#v", nodeConfig.Rules)
	}

	proxy := rules.Methods["hyperflex.proxySettingPolicy.create"]
	if len(proxy.Rules) != 1 || proxy.Rules[0].Minimum[0].Field != "Port" || proxy.Rules[0].Minimum[0].Value != 1 {
		t.Fatalf("unexpected hyperflex proxy rules: %#v", proxy.Rules)
	}

	softwareVersion := rules.Methods["hyperflex.softwareVersionPolicy.create"]
	if len(softwareVersion.Rules) != 0 {
		t.Fatalf("unexpected hyperflex software version rules: %#v", softwareVersion.Rules)
	}

	sysConfig := rules.Methods["hyperflex.sysConfigPolicy.create"]
	if len(sysConfig.Rules) != 0 {
		t.Fatalf("hyperflex.sysConfigPolicy.create rules = %#v, want no custom rules", sysConfig.Rules)
	}

	vcenter := rules.Methods["hyperflex.vcenterConfigPolicy.create"]
	if len(vcenter.Rules) != 0 {
		t.Fatalf("hyperflex.vcenterConfigPolicy.create rules = %#v, want no custom rules", vcenter.Rules)
	}

	ldap := rules.Methods["iam.ldapPolicy.create"]
	if len(ldap.Rules) != 0 {
		t.Fatalf("unexpected ldap rules: %#v", ldap.Rules)
	}

	recoverySchedule := rules.Methods["recovery.scheduleConfigPolicy.create"]
	if len(recoverySchedule.Rules) != 0 {
		t.Fatalf("unexpected recovery schedule rules: %#v", recoverySchedule.Rules)
	}

	scheduler := rules.Methods["scheduler.schedulePolicy.create"]
	if len(scheduler.Rules) != 0 {
		t.Fatalf("unexpected scheduler rules: %#v", scheduler.Rules)
	}

	driveSecurity := rules.Methods["storage.driveSecurityPolicy.create"]
	if len(driveSecurity.Rules) != 0 {
		t.Fatalf("unexpected drive security rules: %#v", driveSecurity.Rules)
	}

	iscsiTarget := rules.Methods["vnic.iscsiStaticTargetPolicy.create"]
	if len(iscsiTarget.Rules) != 0 {
		t.Fatalf("vnic.iscsiStaticTargetPolicy.create rules = %#v, want no custom rules", iscsiTarget.Rules)
	}

	extIscsi := rules.Methods["hyperflex.extIscsiStoragePolicy.create"]
	if len(extIscsi.Rules) != 0 {
		t.Fatalf("hyperflex.extIscsiStoragePolicy.create rules = %#v, want no custom rules", extIscsi.Rules)
	}

	extFc := rules.Methods["hyperflex.extFcStoragePolicy.create"]
	if len(extFc.Rules) != 0 {
		t.Fatalf("hyperflex.extFcStoragePolicy.create rules = %#v, want no custom rules", extFc.Rules)
	}
}

func TestBuildRuleCatalogIncludesAdditionalProbeFindingRules(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/recovery/BackupProfiles": {
				"post": {
					OperationID: "CreateRecoveryBackupProfile",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"Organization": {Relationship: true, RelationshipTarget: "organization.Organization"},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/resourcepool/Pools": {
				"post": {
					OperationID: "CreateResourcepoolPool",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"ResourceType": {Type: "string"},
										"ResourcePoolParameters": {
											Type: "object",
											Properties: map[string]*NormalizedSchema{
												"TargetPlatform": {Type: "string"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/server/Diagnostics": {
				"post": {
					OperationID: "CreateServerDiagnostics",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"ComponentList": {Type: "array", Items: &NormalizedSchema{Type: "string"}},
										"Server":        {Relationship: true, RelationshipTarget: "compute.Physical"},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/fcpool/Reservations": {
				"post": {
					OperationID: "CreateFcpoolReservation",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"AllocationType": {Type: "string"},
										"Pool":           {Relationship: true, RelationshipTarget: "fcpool.Pool"},
										"Organization":   {Relationship: true, RelationshipTarget: "organization.Organization"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.Physical":          {Type: "object"},
			"fcpool.Pool":               {Type: "object"},
			"fcpool.Reservation":        {Type: "object"},
			"organization.Organization": {Type: "object"},
			"recovery.BackupProfile":    {Type: "object"},
			"resourcepool.Pool":         {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"inventory.request.create":       {SDKMethod: "inventory.request.create", Resource: "inventory.Request", Descriptor: OperationDescriptor{OperationID: "CreateInventoryRequest", Method: "POST", PathTemplate: "/api/v1/inventory/Requests"}},
			"recovery.backupProfile.create":  {SDKMethod: "recovery.backupProfile.create", Resource: "recovery.BackupProfile", Descriptor: OperationDescriptor{OperationID: "CreateRecoveryBackupProfile", Method: "POST", PathTemplate: "/api/v1/recovery/BackupProfiles"}},
			"recovery.onDemandBackup.create": {SDKMethod: "recovery.onDemandBackup.create", Resource: "recovery.OnDemandBackup", Descriptor: OperationDescriptor{OperationID: "CreateRecoveryOnDemandBackup", Method: "POST", PathTemplate: "/api/v1/recovery/OnDemandBackups"}},
			"resourcepool.pool.create":       {SDKMethod: "resourcepool.pool.create", Resource: "resourcepool.Pool", Descriptor: OperationDescriptor{OperationID: "CreateResourcepoolPool", Method: "POST", PathTemplate: "/api/v1/resourcepool/Pools"}},
			"server.diagnostics.create":      {SDKMethod: "server.diagnostics.create", Resource: "server.Diagnostics", Descriptor: OperationDescriptor{OperationID: "CreateServerDiagnostics", Method: "POST", PathTemplate: "/api/v1/server/Diagnostics"}},
			"uuidpool.pool.create":           {SDKMethod: "uuidpool.pool.create", Resource: "uuidpool.Pool", Descriptor: OperationDescriptor{OperationID: "CreateUuidpoolPool", Method: "POST", PathTemplate: "/api/v1/uuidpool/Pools"}},
			"iqnpool.pool.create":            {SDKMethod: "iqnpool.pool.create", Resource: "iqnpool.Pool", Descriptor: OperationDescriptor{OperationID: "CreateIqnpoolPool", Method: "POST", PathTemplate: "/api/v1/iqnpool/Pools"}},
			"fcpool.pool.create":             {SDKMethod: "fcpool.pool.create", Resource: "fcpool.Pool", Descriptor: OperationDescriptor{OperationID: "CreateFcpoolPool", Method: "POST", PathTemplate: "/api/v1/fcpool/Pools"}},
			"uuidpool.reservation.create":    {SDKMethod: "uuidpool.reservation.create", Resource: "uuidpool.Reservation", Descriptor: OperationDescriptor{OperationID: "CreateUuidpoolReservation", Method: "POST", PathTemplate: "/api/v1/uuidpool/Reservations"}},
			"macpool.reservation.create":     {SDKMethod: "macpool.reservation.create", Resource: "macpool.Reservation", Descriptor: OperationDescriptor{OperationID: "CreateMacpoolReservation", Method: "POST", PathTemplate: "/api/v1/macpool/Reservations"}},
			"ippool.reservation.create":      {SDKMethod: "ippool.reservation.create", Resource: "ippool.Reservation", Descriptor: OperationDescriptor{OperationID: "CreateIppoolReservation", Method: "POST", PathTemplate: "/api/v1/ippool/Reservations"}},
			"iqnpool.reservation.create":     {SDKMethod: "iqnpool.reservation.create", Resource: "iqnpool.Reservation", Descriptor: OperationDescriptor{OperationID: "CreateIqnpoolReservation", Method: "POST", PathTemplate: "/api/v1/iqnpool/Reservations"}},
			"fcpool.reservation.create":      {SDKMethod: "fcpool.reservation.create", Resource: "fcpool.Reservation", Descriptor: OperationDescriptor{OperationID: "CreateFcpoolReservation", Method: "POST", PathTemplate: "/api/v1/fcpool/Reservations"}},
		},
	}

	templates := []RuleTemplate{
		{
			SDKMethod: "inventory.request.create",
			Resource:  "inventory.Request",
			Rules: []SemanticRule{
				NewRequiredRule("Device", "asset.DeviceRegistration"),
			},
		},
		{
			SDKMethod: "recovery.backupProfile.create",
			Resource:  "recovery.BackupProfile",
			Rules: []SemanticRule{
				NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "recovery.onDemandBackup.create",
			Resource:  "recovery.OnDemandBackup",
			Rules: []SemanticRule{
				NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "resourcepool.pool.create",
			Resource:  "resourcepool.Pool",
			Rules: []SemanticRule{
				NewConditionalRequireRule("ResourceType", "Server", FieldRule{Field: "ResourcePoolParameters"}),
			},
		},
		{
			SDKMethod: "server.diagnostics.create",
			Resource:  "server.Diagnostics",
			Rules: []SemanticRule{
				NewRequiredRule("Server", "compute.Physical"),
				NewRequiredRule("ComponentList", "", 1),
			},
		},
		{
			SDKMethod: "uuidpool.pool.create",
			Resource:  "uuidpool.Pool",
			Rules: []SemanticRule{
				NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "iqnpool.pool.create",
			Resource:  "iqnpool.Pool",
			Rules: []SemanticRule{
				NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "fcpool.pool.create",
			Resource:  "fcpool.Pool",
			Rules: []SemanticRule{
				NewRequiredRule("PoolPurpose", ""),
			},
		},
	}

	for _, sdkMethod := range []string{
		"uuidpool.reservation.create",
		"macpool.reservation.create",
		"ippool.reservation.create",
		"iqnpool.reservation.create",
		"fcpool.reservation.create",
	} {
		resource := catalog.Methods[sdkMethod].Resource
		poolTarget := strings.TrimSuffix(resource, ".Reservation") + ".Pool"
		rules := []SemanticRule{
			NewOneOfRule("AllocationType", "Pool"),
			NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: poolTarget}),
			NewConditionalForbidRule("AllocationType", "static", "Pool"),
		}
		if sdkMethod == "ippool.reservation.create" || sdkMethod == "macpool.reservation.create" {
			rules = append(rules[:2], append([]SemanticRule{NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Identity"})}, rules[2:]...)...)
		}
		if sdkMethod == "fcpool.reservation.create" {
			rules = append([]SemanticRule{NewRequiredRule("Organization", "organization.Organization")}, rules...)
		}
		templates = append(templates, RuleTemplate{
			SDKMethod: sdkMethod,
			Resource:  resource,
			Rules:     rules,
		})
	}

	rules, err := BuildRuleCatalog(spec, catalog, templates)
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	for _, sdkMethod := range []string{
		"uuidpool.reservation.create",
		"macpool.reservation.create",
		"ippool.reservation.create",
		"iqnpool.reservation.create",
		"fcpool.reservation.create",
	} {
		got := rules.Methods[sdkMethod].Rules
		offset := 0
		if sdkMethod == "fcpool.reservation.create" {
			if len(got) != 4 {
				t.Fatalf("%s rules = %#v, want four rules", sdkMethod, got)
			}
			if got[0].Kind != "required" || len(got[0].Require) != 1 || got[0].Require[0].Field != "Organization" || got[0].Require[0].Target != "organization.Organization" {
				t.Fatalf("unexpected reservation organization rule for %s: %#v", sdkMethod, got[0])
			}
			offset = 1
		} else if sdkMethod == "ippool.reservation.create" || sdkMethod == "macpool.reservation.create" {
			if len(got) != 4 {
				t.Fatalf("%s rules = %#v, want four rules", sdkMethod, got)
			}
		} else if len(got) != 3 {
			t.Fatalf("%s rules = %#v, want three rules", sdkMethod, got)
		}
		if got[offset].Kind != "one_of" || len(got[offset].RequireAny) != 2 || got[offset].RequireAny[0].Field != "AllocationType" || got[offset].RequireAny[1].Field != "Pool" {
			t.Fatalf("unexpected reservation one-of rule for %s: %#v", sdkMethod, got[offset])
		}
		if got[offset+1].When == nil || got[offset+1].When.Field != "AllocationType" || got[offset+1].When.Equals != "dynamic" || len(got[offset+1].Require) != 1 || got[offset+1].Require[0].Field != "Pool" {
			t.Fatalf("unexpected reservation dynamic rule for %s: %#v", sdkMethod, got[offset+1])
		}
		staticIndex := offset + 2
		if sdkMethod == "ippool.reservation.create" || sdkMethod == "macpool.reservation.create" {
			if got[offset+2].When == nil || got[offset+2].When.Field != "AllocationType" || got[offset+2].When.Equals != "dynamic" || len(got[offset+2].Require) != 1 || got[offset+2].Require[0].Field != "Identity" {
				t.Fatalf("unexpected reservation dynamic identity rule for %s: %#v", sdkMethod, got[offset+2])
			}
			staticIndex++
		}
		if got[staticIndex].When == nil || got[staticIndex].When.Field != "AllocationType" || got[staticIndex].When.Equals != "static" || !reflect.DeepEqual(got[staticIndex].Forbid, []string{"Pool"}) {
			t.Fatalf("unexpected reservation static forbid rule for %s: %#v", sdkMethod, got[staticIndex])
		}
	}

	if got := rules.Methods["recovery.backupProfile.create"].Rules; len(got) != 1 {
		t.Fatalf("recovery.backupProfile.create rules = %#v, want one rule", got)
	} else if got[0].Kind != "required" || len(got[0].Require) != 1 || got[0].Require[0].Field != "Organization" || got[0].Require[0].Target != "organization.Organization" {
		t.Fatalf("unexpected backup profile rule: %#v", got[0])
	}

	if got := rules.Methods["resourcepool.pool.create"].Rules; len(got) != 1 {
		t.Fatalf("resourcepool.pool.create rules = %#v, want one rule", got)
	} else {
		if got[0].When == nil || got[0].When.Field != "ResourceType" || got[0].When.Equals != "Server" || len(got[0].Require) != 1 || got[0].Require[0].Field != "ResourcePoolParameters" {
			t.Fatalf("unexpected resource pool parameters rule: %#v", got[0])
		}
	}

	if got := rules.Methods["server.diagnostics.create"].Rules; len(got) != 2 {
		t.Fatalf("server.diagnostics.create rules = %#v, want two rules", got)
	} else {
		if got[0].Kind != "required" || len(got[0].Require) != 1 || got[0].Require[0].Field != "Server" || got[0].Require[0].Target != "compute.Physical" {
			t.Fatalf("unexpected server diagnostics server rule: %#v", got[0])
		}
		if got[1].Kind != "required" || len(got[1].Require) != 1 || got[1].Require[0].Field != "ComponentList" {
			t.Fatalf("unexpected server diagnostics component rule: %#v", got[1])
		}
	}
}

func TestBuildRuleCatalogAllowsBodyScopedCustomRules(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/ippool/Pools": {
				"post": {
					OperationID: "CreateIppoolPool",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Circular: "ippool.Pool",
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"ippool.Pool": {
				Type: "object",
				Properties: map[string]*NormalizedSchema{
					"Name":       {Type: "string"},
					"IpV4Blocks": {Type: "array", Items: &NormalizedSchema{Type: "object"}},
				},
			},
		},
	}
	catalog, err := BuildSDKCatalog(spec)
	if err != nil {
		t.Fatalf("BuildSDKCatalog() error = %v", err)
	}
	templates := []RuleTemplate{
		{
			SDKMethod: "ippool.pools.create",
			Resource:  "ippool.Pool",
			Rules: []SemanticRule{
				NewCustomRule(CustomRule{Field: ".", Validator: "ippool_ipv4_blocks_require_config"}),
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, templates)
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}
	method, ok := rules.Methods["ippool.pools.create"]
	if !ok {
		t.Fatalf("rules missing ippool.pools.create")
	}
	if len(method.Rules) != 1 || len(method.Rules[0].Custom) != 1 || method.Rules[0].Custom[0].Field != "." {
		t.Fatalf("body-scoped custom rule was not preserved: %#v", method.Rules)
	}
}

func TestBuildRuleCatalogPropagatesRulesAcrossWriteAliasesForSameOperation(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-16",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/iam/GuestAccessSettings": {
				"post": {
					OperationID: "CreateIamGuestAccessSettings",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"AllowedDomainNames":          {Type: "array", Items: &NormalizedSchema{Type: "string"}},
										"MaxGuestAccessLinkShelfLife": {Type: "integer"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"iam.GuestAccessSettings": {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"iam.guestAccessSetting.create": {
				SDKMethod: "iam.guestAccessSetting.create",
				Resource:  "iam.GuestAccessSettings",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateIamGuestAccessSettings",
					Method:       "POST",
					PathTemplate: "/api/v1/iam/GuestAccessSettings",
				},
			},
			"iam.guestAccessSettings.create": {
				SDKMethod: "iam.guestAccessSettings.create",
				Resource:  "iam.GuestAccessSettings",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateIamGuestAccessSettings",
					Method:       "POST",
					PathTemplate: "/api/v1/iam/GuestAccessSettings",
				},
			},
		},
	}

	templates := []RuleTemplate{
		{
			SDKMethod: "iam.guestAccessSetting.create",
			Resource:  "iam.GuestAccessSettings",
			Rules: []SemanticRule{
				NewRequiredRule("AllowedDomainNames", "", 1),
				NewMinimumRule(MinimumRule{Field: "MaxGuestAccessLinkShelfLife", Value: 86400}),
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, templates)
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	for _, sdkMethod := range []string{"iam.guestAccessSetting.create", "iam.guestAccessSettings.create"} {
		got, ok := rules.Methods[sdkMethod]
		if !ok {
			t.Fatalf("rules missing %s", sdkMethod)
		}
		if len(got.Rules) != 2 {
			t.Fatalf("%s rules = %#v, want two rules", sdkMethod, got.Rules)
		}
	}
}

func TestBuildRuleCatalogResolvesSingularTemplateMethodsToPluralSDKMethods(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-16",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/cond/ThresholdDefinitions": {
				"post": {
					OperationID: "CreateCondThresholdDefinition",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Condition": {Type: "object"},
									},
								},
							},
						},
					},
				},
			},
			"/api/v1/deviceconnector/Policies": {
				"post": {
					OperationID: "CreateDeviceconnectorPolicy",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Organization": {Relationship: true, RelationshipTarget: "organization.Organization"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"cond.ThresholdDefinition":  {Type: "object"},
			"deviceconnector.Policy":    {Type: "object"},
			"organization.Organization": {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"cond.thresholdDefinitions.create": {
				SDKMethod: "cond.thresholdDefinitions.create",
				Resource:  "cond.ThresholdDefinition",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateCondThresholdDefinition",
					Method:       "POST",
					PathTemplate: "/api/v1/cond/ThresholdDefinitions",
				},
			},
			"deviceconnector.policies.create": {
				SDKMethod: "deviceconnector.policies.create",
				Resource:  "deviceconnector.Policy",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateDeviceconnectorPolicy",
					Method:       "POST",
					PathTemplate: "/api/v1/deviceconnector/Policies",
				},
			},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog, testIntersightRuleTemplates())
	if err != nil {
		t.Fatalf("BuildRuleCatalog() error = %v", err)
	}

	threshold, ok := rules.Methods["cond.thresholdDefinitions.create"]
	if !ok {
		t.Fatalf("rules missing cond.thresholdDefinitions.create")
	}
	if len(threshold.Rules) != 1 || len(threshold.Rules[0].Require) != 1 || threshold.Rules[0].Require[0].Field != "Condition" {
		t.Fatalf("unexpected threshold rules: %#v", threshold.Rules)
	}

	deviceconnector, ok := rules.Methods["deviceconnector.policies.create"]
	if !ok {
		t.Fatalf("rules missing deviceconnector.policies.create")
	}
	if len(deviceconnector.Rules) != 1 || len(deviceconnector.Rules[0].Require) != 1 {
		t.Fatalf("unexpected deviceconnector rules: %#v", deviceconnector.Rules)
	}
	if deviceconnector.Rules[0].Require[0].Field != "Organization" || deviceconnector.Rules[0].Require[0].Target != "organization.Organization" {
		t.Fatalf("unexpected deviceconnector organization rule: %#v", deviceconnector.Rules[0])
	}
}

func TestValidateRuleCatalogAgainstArtifactsRejectsCanonicalResourceMismatch(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Properties: map[string]*NormalizedSchema{
										"Name": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"example.Widget":      {Type: "object"},
			"example.OtherWidget": {Type: "object"},
		},
	}
	catalog := SDKCatalog{
		Metadata: spec.Metadata,
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
		},
	}
	rules := RuleCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]MethodRules{
			"example.widget.create": {
				SDKMethod:   "example.widget.create",
				OperationID: "CreateExampleWidget",
				Resource:    "example.OtherWidget",
			},
		},
	}

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules, []RuleTemplate{
		{
			SDKMethod: "example.widget.create",
			Resource:  "example.Widget",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not match sdk catalog resource") {
		t.Fatalf("ValidateRuleCatalogAgainstArtifacts() error = %v, want resource mismatch", err)
	}
}

func TestValidateMethodRulesAcceptsOneOfRules(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Primary":   {Type: "string"},
										"Secondary": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit": {},
			"example.Widget":   {Type: "object"},
		},
	}

	bodySchema := &NormalizedSchema{
		Type: "object",
		Properties: map[string]*NormalizedSchema{
			"Primary":   {Type: "string"},
			"Secondary": {Type: "string"},
		},
	}

	methodRules := MethodRules{
		SDKMethod:   "example.widget.create",
		OperationID: "CreateExampleWidget",
		Resource:    "example.Widget",
		Rules: []SemanticRule{
			{
				Kind:       "one_of",
				RequireAny: []FieldRule{{Field: "Primary"}, {Field: "Secondary"}},
			},
		},
	}

	if err := validateMethodRules(spec, "example.widget.create", methodRules, bodySchema); err != nil {
		t.Fatalf("validateMethodRules() error = %v", err)
	}
}

func TestValidateRuleCatalogAgainstArtifactsRejectsRequiredRuleKind(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Name": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit": {},
			"example.Widget":   {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"example.widget.create": {
				SDKMethod: "example.widget.create",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
		},
	}

	rules := RuleCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]MethodRules{
			"example.widget.create": {
				SDKMethod:   "example.widget.create",
				OperationID: "CreateExampleWidget",
				Resource:    "example.Widget",
				Rules: []SemanticRule{
					{
						Kind:    "required",
						Require: []FieldRule{{Field: "MissingField"}},
					},
				},
			},
		},
	}

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules, []RuleTemplate{
		{
			SDKMethod: "example.widget.create",
			Resource:  "example.Widget",
			Rules: []SemanticRule{
				{
					Kind:    "required",
					Require: []FieldRule{{Field: "MissingField"}},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown required field") {
		t.Fatalf("ValidateRuleCatalogAgainstArtifacts() error = %v, want unknown required field failure", err)
	}
}

func TestValidateRuleCatalogAgainstArtifactsRejectsMismatchedRelationshipTarget(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
		Paths: map[string]map[string]NormalizedOperation{
			"/api/v1/example/Widgets": {
				"post": {
					OperationID: "CreateExampleWidget",
					RequestBody: &NormalizedRequestBody{
						Content: map[string]NormalizedMediaContent{
							"application/json": {
								Schema: &NormalizedSchema{
									Type: "object",
									Properties: map[string]*NormalizedSchema{
										"Organization": {
											Relationship:       true,
											RelationshipTarget: "organization.Organization",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Schemas: map[string]NormalizedSchema{
			"compute.RackUnit":          {},
			"example.Widget":            {Type: "object"},
			"organization.Organization": {Type: "object"},
			"other.Target":              {Type: "object"},
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"example.widget.create": {
				SDKMethod: "example.widget.create",
				Descriptor: OperationDescriptor{
					OperationID:  "CreateExampleWidget",
					Method:       "POST",
					PathTemplate: "/api/v1/example/Widgets",
				},
			},
		},
	}

	rules := RuleCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]MethodRules{
			"example.widget.create": {
				SDKMethod:   "example.widget.create",
				OperationID: "CreateExampleWidget",
				Resource:    "example.Widget",
				Rules: []SemanticRule{
					{
						Kind:    "conditional",
						When:    &RuleCondition{Field: "Organization", Equals: true},
						Require: []FieldRule{{Field: "Organization", Target: "other.Target"}},
					},
				},
			},
		},
	}

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules, []RuleTemplate{
		{
			SDKMethod: "example.widget.create",
			Resource:  "example.Widget",
			Rules: []SemanticRule{
				{
					Kind:    "conditional",
					When:    &RuleCondition{Field: "Organization", Equals: true},
					Require: []FieldRule{{Field: "Organization", Target: "other.Target"}},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not match embedded spec target") {
		t.Fatalf("ValidateRuleCatalogAgainstArtifacts() error = %v, want relationship target failure", err)
	}
}
