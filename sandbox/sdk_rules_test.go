package sandbox

import (
	"testing"

	"github.com/mimaurer/intersight-mcp/internal/contracts"
)

func TestValidateSemanticRulesTypedIssues(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Require: []contracts.FieldRule{{Field: "Organization"}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Require: []contracts.FieldRule{{Field: "Tags", MinCount: 2}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Forbid: []string{"Deprecated"},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Minimum: []contracts.MinimumRule{{Field: "Priority", Value: 10}},
						},
						{
							Kind:       "conditional",
							When:       &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							RequireAny: []contracts.FieldRule{{Field: "Primary"}, {Field: "Secondary"}},
						},
						{
							Kind:        "conditional",
							When:        &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							RequireEach: []contracts.FieldRule{{Field: "Items[].Name"}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Maximum: []contracts.LengthRule{{Field: "Username", Value: 4}},
						},
						{
							Kind:    "conditional",
							When:    &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Pattern: []contracts.PatternRule{{Field: "Slug", Value: "^[a-z]+$"}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Future: []contracts.TimeRule{{Field: "StartsAt"}},
						},
						{
							Kind:     "conditional",
							When:     &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Contains: []contracts.ContainsRule{{Field: "Kinds[]", Value: "gpu"}},
						},
						{
							Kind:   "conditional",
							When:   &contracts.RuleCondition{Field: "Mode", Equals: "fast"},
							Custom: []contracts.CustomRule{{Field: "Filter", Validator: "ldap_filter"}},
						},
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Mode":       "fast",
		"Tags":       []any{"one"},
		"Deprecated": true,
		"Priority":   5,
		"Items":      []any{map[string]any{}, map[string]any{"Name": "ok"}},
		"Username":   "too-long",
		"Slug":       "UPPER",
		"StartsAt":   "2020-01-01T00:00:00Z",
		"Kinds":      []any{"cpu"},
		"Filter":     "uid=user",
	})

	if len(errs) != 11 {
		t.Fatalf("len(errs) = %d, want 11", len(errs))
	}
	assertSemanticIssue(t, errs[0], "Organization", "required")
	assertSemanticIssue(t, errs[1], "Tags", "min_items")
	assertSemanticIssue(t, errs[2], "Deprecated", "forbidden")
	assertSemanticIssue(t, errs[3], "Priority", "minimum")
	assertSemanticIssue(t, errs[4], "Primary|Secondary", "one_of")
	assertSemanticIssue(t, errs[5], "Items[].Name", "required_each")
	assertSemanticIssue(t, errs[6], "Username", "maximum")
	assertSemanticIssue(t, errs[7], "Slug", "pattern")
	assertSemanticIssue(t, errs[8], "StartsAt", "future")
	assertSemanticIssue(t, errs[9], "Kinds[]", "contains")
	assertSemanticIssue(t, errs[10], "Filter", "custom")

	for _, err := range errs {
		if err.Condition == "" {
			t.Fatalf("condition = %q, want non-empty", err.Condition)
		}
	}
}

func TestValidateSemanticRulesOneOfSatisfied(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						{
							Kind:       "one_of",
							RequireAny: []contracts.FieldRule{{Field: "Primary"}, {Field: "Secondary"}},
						},
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Secondary": "value",
	})
	if len(errs) != 0 {
		t.Fatalf("len(errs) = %d, want 0: %#v", len(errs), errs)
	}
}

func TestValidateSemanticRulesCustomProbeValidators(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"example.widget.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewConditionalInCustomRule("Mode", []any{"auto", "on"}, contracts.CustomRule{Field: "ReceiveDirection", Validator: "disabled_string"}),
						contracts.NewCustomRule(contracts.CustomRule{Field: "VlanSettings", Validator: "native_vlan_in_allowed_vlans"}),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("example.widget.create", map[string]any{
		"Mode":             "auto",
		"ReceiveDirection": "Enabled",
		"VlanSettings": map[string]any{
			"AllowedVlans": "2-3",
			"NativeVlan":   1,
		},
	})

	if len(errs) != 2 {
		t.Fatalf("len(errs) = %d, want 2: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "ReceiveDirection", "custom")
	assertSemanticIssue(t, errs[1], "VlanSettings", "custom")
}

func TestValidateSemanticRulesBodyScopedCustomValidators(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"ippool.pools.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewCustomRule(contracts.CustomRule{Field: ".", Validator: "ippool_ipv4_blocks_require_config"}),
					},
				},
				"memory.persistentMemoryPolicies.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewCustomRule(contracts.CustomRule{Field: ".", Validator: "persistent_memory_os_mode"}),
					},
				},
				"iqnpool.pools.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewCustomRule(contracts.CustomRule{Field: ".", Validator: "iqnpool_suffix_blocks_require_suffix"}),
					},
				},
			},
		},
	}

	ipPoolErrs := runtime.validateSemanticRules("ippool.pools.create", map[string]any{
		"IpV4Blocks": []any{
			map[string]any{"From": "10.10.10.10", "Size": 4},
		},
	})
	if len(ipPoolErrs) != 1 {
		t.Fatalf("len(ipPoolErrs) = %d, want 1: %#v", len(ipPoolErrs), ipPoolErrs)
	}
	assertSemanticIssue(t, ipPoolErrs[0], ".", "custom")

	ipPoolOK := runtime.validateSemanticRules("ippool.pools.create", map[string]any{
		"IpV4Config": map[string]any{"Netmask": "255.255.255.0"},
		"IpV4Blocks": []any{
			map[string]any{"From": "10.10.10.10", "Size": 4},
		},
	})
	if len(ipPoolOK) != 0 {
		t.Fatalf("len(ipPoolOK) = %d, want 0: %#v", len(ipPoolOK), ipPoolOK)
	}

	pmemErrs := runtime.validateSemanticRules("memory.persistentMemoryPolicies.create", map[string]any{
		"ManagementMode":   "configured-from-operating-system",
		"RetainNamespaces": false,
	})
	if len(pmemErrs) != 1 {
		t.Fatalf("len(pmemErrs) = %d, want 1: %#v", len(pmemErrs), pmemErrs)
	}
	assertSemanticIssue(t, pmemErrs[0], ".", "custom")

	pmemOK := runtime.validateSemanticRules("memory.persistentMemoryPolicies.create", map[string]any{
		"ManagementMode": "configured-from-operating-system",
	})
	if len(pmemOK) != 0 {
		t.Fatalf("len(pmemOK) = %d, want 0: %#v", len(pmemOK), pmemOK)
	}

	iqnErrs := runtime.validateSemanticRules("iqnpool.pools.create", map[string]any{
		"IqnSuffixBlocks": []any{
			map[string]any{"From": 10, "To": 20},
		},
	})
	if len(iqnErrs) != 1 {
		t.Fatalf("len(iqnErrs) = %d, want 1: %#v", len(iqnErrs), iqnErrs)
	}
	assertSemanticIssue(t, iqnErrs[0], ".", "custom")

	iqnOK := runtime.validateSemanticRules("iqnpool.pools.create", map[string]any{
		"IqnSuffixBlocks": []any{
			map[string]any{"From": 10, "To": 20, "Suffix": "host"},
		},
	})
	if len(iqnOK) != 0 {
		t.Fatalf("len(iqnOK) = %d, want 0: %#v", len(iqnOK), iqnOK)
	}
}

func TestValidateSemanticRulesDynamicReservationsRequireIdentity(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"ippool.reservations.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewConditionalRequireRule("AllocationType", "dynamic", contracts.FieldRule{Field: "Pool", Target: "ippool.Pool"}),
						contracts.NewConditionalRequireRule("AllocationType", "dynamic", contracts.FieldRule{Field: "Identity"}),
					},
				},
				"macpool.reservations.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewConditionalRequireRule("AllocationType", "dynamic", contracts.FieldRule{Field: "Pool", Target: "macpool.Pool"}),
						contracts.NewConditionalRequireRule("AllocationType", "dynamic", contracts.FieldRule{Field: "Identity"}),
					},
				},
			},
		},
	}

	for _, sdkMethod := range []string{"ippool.reservations.create", "macpool.reservations.create"} {
		errs := runtime.validateSemanticRules(sdkMethod, map[string]any{
			"AllocationType": "dynamic",
			"Pool":           map[string]any{"Moid": "1"},
		})
		if len(errs) != 1 {
			t.Fatalf("%s len(errs) = %d, want 1: %#v", sdkMethod, len(errs), errs)
		}
		assertSemanticIssue(t, errs[0], "Identity", "required")
	}
}

func TestValidateSemanticRulesNetFlowValidators(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"fabric.netFlowRecords.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("RecordType", ""),
						contracts.NewRequiredRule("FlowNonKey", ""),
						contracts.NewCustomRule(contracts.CustomRule{Field: "RecordType", Validator: "netflow_record_type"}),
						contracts.NewConditionalRequireRule("RecordType", "IPv4", contracts.FieldRule{Field: "Ipv4FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "IPv4", "Ipv6FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "IPv4", "L2FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "IPv4", contracts.CustomRule{Field: "Ipv4FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewConditionalRequireRule("RecordType", "IPv6", contracts.FieldRule{Field: "Ipv6FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "IPv6", "Ipv4FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "IPv6", "L2FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "IPv6", contracts.CustomRule{Field: "Ipv6FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewConditionalRequireRule("RecordType", "L2", contracts.FieldRule{Field: "L2FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "L2", "Ipv4FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "L2", "Ipv6FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "L2", contracts.CustomRule{Field: "L2FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewCustomRule(contracts.CustomRule{Field: "FlowNonKey", Validator: "netflow_non_key_fields"}),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("fabric.netFlowRecords.create", map[string]any{
		"RecordType":  "IPv4",
		"Ipv4FlowKey": map[string]any{"SourceIpAddress": false},
		"FlowNonKey":  map[string]any{"PacketCounters": false},
	})

	if len(errs) != 2 {
		t.Fatalf("len(errs) = %d, want 2: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "Ipv4FlowKey", "custom")
	assertSemanticIssue(t, errs[1], "FlowNonKey", "custom")
}

func TestValidateSemanticRulesNetFlowSelectedFamilyRequiredAndMismatchedFamiliesForbidden(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"fabric.netFlowRecords.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("RecordType", ""),
						contracts.NewRequiredRule("FlowNonKey", ""),
						contracts.NewCustomRule(contracts.CustomRule{Field: "RecordType", Validator: "netflow_record_type"}),
						contracts.NewConditionalRequireRule("RecordType", "IPv4", contracts.FieldRule{Field: "Ipv4FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "IPv4", "Ipv6FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "IPv4", "L2FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "IPv4", contracts.CustomRule{Field: "Ipv4FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewConditionalRequireRule("RecordType", "IPv6", contracts.FieldRule{Field: "Ipv6FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "IPv6", "Ipv4FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "IPv6", "L2FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "IPv6", contracts.CustomRule{Field: "Ipv6FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewConditionalRequireRule("RecordType", "L2", contracts.FieldRule{Field: "L2FlowKey"}),
						contracts.NewConditionalForbidRule("RecordType", "L2", "Ipv4FlowKey"),
						contracts.NewConditionalForbidRule("RecordType", "L2", "Ipv6FlowKey"),
						contracts.NewConditionalCustomRule("RecordType", "L2", contracts.CustomRule{Field: "L2FlowKey", Validator: "netflow_key_fields"}),
						contracts.NewCustomRule(contracts.CustomRule{Field: "FlowNonKey", Validator: "netflow_non_key_fields"}),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("fabric.netFlowRecords.create", map[string]any{
		"RecordType": "IPv4",
		"FlowNonKey": map[string]any{"ByteCounters": true},
		"Ipv6FlowKey": map[string]any{
			"SourceIpAddress": true,
		},
	})

	if len(errs) != 2 {
		t.Fatalf("len(errs) = %d, want 2: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "Ipv4FlowKey", "required")
	assertSemanticIssue(t, errs[1], "Ipv6FlowKey", "forbidden")
}

func TestValidateSemanticRulesNetFlowInvalidRecordType(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"fabric.netFlowRecords.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("RecordType", ""),
						contracts.NewCustomRule(contracts.CustomRule{Field: "RecordType", Validator: "netflow_record_type"}),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("fabric.netFlowRecords.create", map[string]any{
		"RecordType": "Invalid",
	})

	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "RecordType", "custom")
}

func TestValidateSemanticRulesHighConfidenceReadOnlyAndMissingPropertyForbids(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"fabric.vlans.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewForbidRule("VlanSet"),
					},
				},
				"firmware.unsupportedVersionUpgrades.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewForbidRule("UpgradeType"),
					},
				},
				"iam.apiKeys.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewForbidRule("User"),
						contracts.NewForbidRule("Permission"),
					},
				},
			},
		},
	}

	errs := runtime.validateSemanticRules("fabric.vlans.create", map[string]any{
		"VlanSet": map[string]any{"Moid": "123"},
	})
	if len(errs) != 1 {
		t.Fatalf("len(errs) for vlan = %d, want 1: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "VlanSet", "forbidden")

	errs = runtime.validateSemanticRules("firmware.unsupportedVersionUpgrades.create", map[string]any{
		"UpgradeType": "direct_upgrade",
	})
	if len(errs) != 1 {
		t.Fatalf("len(errs) for unsupported upgrade = %d, want 1: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "UpgradeType", "forbidden")

	errs = runtime.validateSemanticRules("iam.apiKeys.create", map[string]any{
		"User":       map[string]any{"Moid": "123"},
		"Permission": map[string]any{"Moid": "456"},
	})
	if len(errs) != 2 {
		t.Fatalf("len(errs) for api key = %d, want 2: %#v", len(errs), errs)
	}
	assertSemanticIssue(t, errs[0], "User", "forbidden")
	assertSemanticIssue(t, errs[1], "Permission", "forbidden")
}

func TestValidateSemanticRulesHighConfidenceProbeFindings(t *testing.T) {
	t.Parallel()

	runtime := &sdkRuntime{
		rules: contracts.RuleCatalog{
			Methods: map[string]contracts.MethodRules{
				"fabric.netFlowMonitors.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("NetFlowPolicy", "fabric.NetFlowPolicy"),
						contracts.NewRequiredRule("FlowRecord", "fabric.NetFlowRecord"),
						contracts.NewRequiredRule("FlowExporters", "fabric.NetFlowExporter", 1),
					},
				},
				"fabric.switchProfiles.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("SwitchClusterProfile", "fabric.SwitchClusterProfile"),
					},
				},
				"fabric.spanSourceVnicEthIfs.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewRequiredRule("SpanSession", "fabric.SpanSession"),
						contracts.NewForbidRule("Name"),
					},
				},
				"iam.certificates.create": {
					Rules: []contracts.SemanticRule{
						contracts.NewOneOfRule("Certificate", "CertificateRequest"),
					},
				},
			},
		},
	}

	monitorErrs := runtime.validateSemanticRules("fabric.netFlowMonitors.create", map[string]any{
		"NetFlowPolicy": map[string]any{"Moid": "1"},
		"FlowExporters": []any{},
	})
	if len(monitorErrs) != 2 {
		t.Fatalf("len(monitorErrs) = %d, want 2: %#v", len(monitorErrs), monitorErrs)
	}
	assertSemanticIssue(t, monitorErrs[0], "FlowRecord", "required")
	assertSemanticIssue(t, monitorErrs[1], "FlowExporters", "min_items")

	switchErrs := runtime.validateSemanticRules("fabric.switchProfiles.create", map[string]any{
		"Name": "switch-a",
	})
	if len(switchErrs) != 1 {
		t.Fatalf("len(switchErrs) = %d, want 1: %#v", len(switchErrs), switchErrs)
	}
	assertSemanticIssue(t, switchErrs[0], "SwitchClusterProfile", "required")

	spanErrs := runtime.validateSemanticRules("fabric.spanSourceVnicEthIfs.create", map[string]any{
		"Name": "read-only-name",
	})
	if len(spanErrs) != 2 {
		t.Fatalf("len(spanErrs) = %d, want 2: %#v", len(spanErrs), spanErrs)
	}
	assertSemanticIssue(t, spanErrs[0], "SpanSession", "required")
	assertSemanticIssue(t, spanErrs[1], "Name", "forbidden")

	certErrs := runtime.validateSemanticRules("iam.certificates.create", map[string]any{})
	if len(certErrs) != 1 {
		t.Fatalf("len(certErrs) = %d, want 1: %#v", len(certErrs), certErrs)
	}
	assertSemanticIssue(t, certErrs[0], "Certificate|CertificateRequest", "one_of")
}

func assertSemanticIssue(t *testing.T, err dryRunValidationError, path, issueType string) {
	t.Helper()

	if err.Path != path {
		t.Fatalf("path = %q, want %q", err.Path, path)
	}
	if err.Type != issueType {
		t.Fatalf("type = %q, want %q", err.Type, issueType)
	}
	if err.Source != validationSourceRules {
		t.Fatalf("source = %q, want %q", err.Source, validationSourceRules)
	}
}
