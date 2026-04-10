package contracts

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildRuleCatalogIncludesPostWriteMethodsForPhaseFourResources(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
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

	rules, err := BuildRuleCatalog(spec, catalog)
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

	rules, err := BuildRuleCatalog(spec, catalog)
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

func TestBuildRuleCatalogIncludesPolicyCreateRulesFromProbeFindings(t *testing.T) {
	t.Parallel()

	spec := NormalizedSpec{
		Metadata: ArtifactSourceMetadata{
			PublishedVersion: "1.0.0-test",
			SourceURL:        "https://example.com/spec",
			SHA256:           "abc123",
			RetrievalDate:    "2026-04-08",
		},
	}

	catalog := SDKCatalog{
		Metadata: spec.Metadata,
		Methods: map[string]SDKMethod{
			"aaa.retentionPolicy.create":             {SDKMethod: "aaa.retentionPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateAaaRetentionPolicy", Method: "POST"}},
			"access.policy.create":                   {SDKMethod: "access.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateAccessPolicy", Method: "POST"}},
			"appliance.dataExportPolicy.create":      {SDKMethod: "appliance.dataExportPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateApplianceDataExportPolicy", Method: "POST"}},
			"hyperflex.extFcStoragePolicy.create":    {SDKMethod: "hyperflex.extFcStoragePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexExtFcStoragePolicy", Method: "POST"}},
			"hyperflex.extIscsiStoragePolicy.create": {SDKMethod: "hyperflex.extIscsiStoragePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexExtIscsiStoragePolicy", Method: "POST"}},
			"hyperflex.localCredentialPolicy.create": {SDKMethod: "hyperflex.localCredentialPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexLocalCredentialPolicy", Method: "POST"}},
			"hyperflex.nodeConfigPolicy.create":      {SDKMethod: "hyperflex.nodeConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexNodeConfigPolicy", Method: "POST"}},
			"hyperflex.proxySettingPolicy.create":    {SDKMethod: "hyperflex.proxySettingPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexProxySettingPolicy", Method: "POST"}},
			"hyperflex.softwareVersionPolicy.create": {SDKMethod: "hyperflex.softwareVersionPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexSoftwareVersionPolicy", Method: "POST"}},
			"hyperflex.sysConfigPolicy.create":       {SDKMethod: "hyperflex.sysConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexSysConfigPolicy", Method: "POST"}},
			"hyperflex.vcenterConfigPolicy.create":   {SDKMethod: "hyperflex.vcenterConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateHyperflexVcenterConfigPolicy", Method: "POST"}},
			"iam.ldapPolicy.create":                  {SDKMethod: "iam.ldapPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateIamLdapPolicy", Method: "POST"}},
			"ntp.policy.create":                      {SDKMethod: "ntp.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateNtpPolicy", Method: "POST"}},
			"recovery.scheduleConfigPolicy.create":   {SDKMethod: "recovery.scheduleConfigPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateRecoveryScheduleConfigPolicy", Method: "POST"}},
			"scheduler.schedulePolicy.create":        {SDKMethod: "scheduler.schedulePolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateSchedulerSchedulePolicy", Method: "POST"}},
			"smtp.policy.create":                     {SDKMethod: "smtp.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateSmtpPolicy", Method: "POST"}},
			"smtp.policyTest.create":                 {SDKMethod: "smtp.policyTest.create", Descriptor: OperationDescriptor{OperationID: "CreateSmtpPolicyTest", Method: "POST"}},
			"storage.driveSecurityPolicy.create":     {SDKMethod: "storage.driveSecurityPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateStorageDriveSecurityPolicy", Method: "POST"}},
			"syslog.policy.create":                   {SDKMethod: "syslog.policy.create", Descriptor: OperationDescriptor{OperationID: "CreateSyslogPolicy", Method: "POST"}},
			"vnic.iscsiAdapterPolicy.create":         {SDKMethod: "vnic.iscsiAdapterPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateVnicIscsiAdapterPolicy", Method: "POST"}},
			"vnic.iscsiStaticTargetPolicy.create":    {SDKMethod: "vnic.iscsiStaticTargetPolicy.create", Descriptor: OperationDescriptor{OperationID: "CreateVnicIscsiStaticTargetPolicy", Method: "POST"}},
		},
	}

	rules, err := BuildRuleCatalog(spec, catalog)
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

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules)
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

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules)
	if err == nil || !strings.Contains(err.Error(), "unsupported rule kind") {
		t.Fatalf("ValidateRuleCatalogAgainstArtifacts() error = %v, want unsupported required rule failure", err)
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

	err := ValidateRuleCatalogAgainstArtifacts(spec, catalog, rules)
	if err == nil || !strings.Contains(err.Error(), "does not match embedded spec target") {
		t.Fatalf("ValidateRuleCatalogAgainstArtifacts() error = %v, want relationship target failure", err)
	}
}
