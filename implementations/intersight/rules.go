package intersight

import "github.com/mimaurer/intersight-mcp/internal/contracts"

type RuleTemplate = contracts.RuleTemplate
type SemanticRule = contracts.SemanticRule
type FieldRule = contracts.FieldRule
type MinimumRule = contracts.MinimumRule
type LengthRule = contracts.LengthRule
type PatternRule = contracts.PatternRule
type ContainsRule = contracts.ContainsRule
type CustomRule = contracts.CustomRule

func RuleTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			SDKMethod: "aaa.retentionPolicy.create",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("RetentionPeriod", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "aaa.retentionPolicy.post",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("RetentionPeriod", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "aaa.retentionPolicy.update",
			Resource:  "aaa.RetentionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("RetentionPeriod", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "RetentionPeriod", Value: 6}),
			},
		},
		{
			SDKMethod: "access.policy.create",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("AddressType", ""),
				contracts.NewRequiredRule("ConfigurationType", ""),
				contracts.NewConditionalRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				contracts.NewConditionalMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "access.policy.post",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("AddressType", ""),
				contracts.NewRequiredRule("ConfigurationType", ""),
				contracts.NewConditionalRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				contracts.NewConditionalMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "access.policy.update",
			Resource:  "access.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("AddressType", ""),
				contracts.NewRequiredRule("ConfigurationType", ""),
				contracts.NewConditionalRequireRule("ConfigurationType.ConfigureInband", true, FieldRule{Field: "InbandIpPool", Target: "ippool.Pool"}),
				contracts.NewConditionalMinimumRule("ConfigurationType.ConfigureInband", true, MinimumRule{Field: "InbandVlan", Value: 4}),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.create",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.post",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "adapter.configPolicy.update",
			Resource:  "adapter.ConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Settings", "", 1),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.create",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				contracts.NewForbidRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.post",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				contracts.NewForbidRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.update",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				contracts.NewForbidRule("Name"),
			},
		},
		{
			SDKMethod: "appliance.dataExportPolicy.patch",
			Resource:  "appliance.DataExportPolicy",
			Rules: []SemanticRule{
				contracts.NewForbidRule("Name"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.create",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("StartDate", ""),
				contracts.NewOneOfRule("Entity", "AlarmRules"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.post",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("StartDate", ""),
				contracts.NewOneOfRule("Entity", "AlarmRules"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.update",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("Entity", "AlarmRules"),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.create",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.post",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "comm.httpProxyPolicy.update",
			Resource:  "comm.HttpProxyPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
			},
		},
		{
			SDKMethod: "comm.tagDefinition.create",
			Resource:  "comm.TagDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Key", ""),
			},
		},
		{
			SDKMethod: "comm.tagDefinition.post",
			Resource:  "comm.TagDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Key", ""),
			},
		},
		{
			SDKMethod: "comm.tagDefinition.update",
			Resource:  "comm.TagDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Key", ""),
			},
		},
		{
			SDKMethod: "fcpool.pool.create",
			Resource:  "fcpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PoolPurpose", ""),
			},
		},
		{
			SDKMethod: "fcpool.pool.post",
			Resource:  "fcpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PoolPurpose", ""),
			},
		},
		{
			SDKMethod: "fcpool.pool.update",
			Resource:  "fcpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PoolPurpose", ""),
			},
		},
		{
			SDKMethod: "fcpool.reservation.create",
			Resource:  "fcpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Organization", "organization.Organization"),
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "fcpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "fcpool.reservation.post",
			Resource:  "fcpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Organization", "organization.Organization"),
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "fcpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "fcpool.reservation.update",
			Resource:  "fcpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Organization", "organization.Organization"),
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "fcpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "inventory.request.create",
			Resource:  "inventory.Request",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Device", "asset.DeviceRegistration"),
			},
		},
		{
			SDKMethod: "inventory.request.post",
			Resource:  "inventory.Request",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Device", "asset.DeviceRegistration"),
			},
		},
		{
			SDKMethod: "inventory.request.update",
			Resource:  "inventory.Request",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Device", "asset.DeviceRegistration"),
			},
		},
		{
			SDKMethod: "ippool.reservation.create",
			Resource:  "ippool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "ippool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "ippool.reservation.post",
			Resource:  "ippool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "ippool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "ippool.reservation.update",
			Resource:  "ippool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "ippool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "iqnpool.pool.create",
			Resource:  "iqnpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "iqnpool.pool.post",
			Resource:  "iqnpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "iqnpool.pool.update",
			Resource:  "iqnpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "iqnpool.reservation.create",
			Resource:  "iqnpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "iqnpool.reservation.post",
			Resource:  "iqnpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "iqnpool.reservation.update",
			Resource:  "iqnpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "macpool.reservation.create",
			Resource:  "macpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "macpool.reservation.post",
			Resource:  "macpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "macpool.reservation.update",
			Resource:  "macpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.create",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.post",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.update",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
			},
		},
		{
			SDKMethod: "auditd.policy.create",
			Resource:  "auditd.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "vnic.ethIf.create",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				contracts.NewRequiredRule("EthAdapterPolicy", "vnic.EthAdapterPolicy"),
				contracts.NewRequiredRule("EthQosPolicy", "vnic.EthQosPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				contracts.NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				contracts.NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				contracts.NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				contracts.NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.post",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				contracts.NewRequiredRule("EthAdapterPolicy", "vnic.EthAdapterPolicy"),
				contracts.NewRequiredRule("EthQosPolicy", "vnic.EthQosPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				contracts.NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				contracts.NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				contracts.NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				contracts.NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.update",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LanConnectivityPolicy", "vnic.LanConnectivityPolicy"),
				contracts.NewRequiredRule("EthAdapterPolicy", "vnic.EthAdapterPolicy"),
				contracts.NewRequiredRule("EthQosPolicy", "vnic.EthQosPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("FabricEthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy", 1),
				contracts.NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				contracts.NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				contracts.NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				contracts.NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.ethIf.patch",
			Resource:  "vnic.EthIf",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("MacAddressType", "POOL", FieldRule{Field: "MacPool", Target: "macpool.Pool"}),
				contracts.NewConditionalForbidRule("MacAddressType", "POOL", "StaticMacAddress"),
				contracts.NewConditionalRequireRule("MacAddressType", "STATIC", FieldRule{Field: "StaticMacAddress"}),
				contracts.NewConditionalForbidRule("MacAddressType", "STATIC", "MacPool"),
				contracts.NewConditionalInRequireRule("Placement.SwitchId", []any{"A", "B"}, FieldRule{Field: "Placement.AutoSlotId"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.create",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				contracts.NewConditionalRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Static", "IqnPool"),
				contracts.NewConditionalRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.post",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				contracts.NewConditionalRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Static", "IqnPool"),
				contracts.NewConditionalRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.update",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				contracts.NewConditionalRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Static", "IqnPool"),
				contracts.NewConditionalRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.lanConnectivityPolicy.patch",
			Resource:  "vnic.LanConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("IqnAllocationType", "Pool", FieldRule{Field: "IqnPool", Target: "iqnpool.Pool"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Pool", "StaticIqnName"),
				contracts.NewConditionalRequireRule("IqnAllocationType", "Static", FieldRule{Field: "StaticIqnName"}),
				contracts.NewConditionalForbidRule("IqnAllocationType", "Static", "IqnPool"),
				contracts.NewConditionalRequireRule("PlacementMode", "custom", FieldRule{Field: "EthIfs", MinCount: 1, Target: "vnic.EthIf"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.create",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.post",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.update",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.patch",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.create",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				contracts.NewConditionalRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				contracts.NewConditionalMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.post",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				contracts.NewConditionalRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				contracts.NewConditionalMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.update",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				contracts.NewConditionalRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				contracts.NewConditionalMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethAdapterPolicy.patch",
			Resource:  "vnic.EthAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("RssSettings", true, FieldRule{Field: "RssHashSettings"}),
				contracts.NewConditionalRequireRule("EtherChannelPinningEnabled", true, FieldRule{Field: "TxQueueSettings"}),
				contracts.NewConditionalMinimumRule("EtherChannelPinningEnabled", true, MinimumRule{Field: "TxQueueSettings.Count", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.create",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewCustomRule(CustomRule{Field: "VlanSettings", Validator: "native_vlan_in_allowed_vlans"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.post",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewCustomRule(CustomRule{Field: "VlanSettings", Validator: "native_vlan_in_allowed_vlans"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.update",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewCustomRule(CustomRule{Field: "VlanSettings", Validator: "native_vlan_in_allowed_vlans"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.patch",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.create",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.post",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "fabric.macSecPolicy.update",
			Resource:  "fabric.MacSecPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryKeyChain", ""),
			},
		},
		{
			SDKMethod: "fabric.flowControlPolicy.create",
			Resource:  "fabric.FlowControlPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "ReceiveDirection", Validator: "disabled_string"}),
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "SendDirection", Validator: "disabled_string"}),
			},
		},
		{
			SDKMethod: "fabric.flowControlPolicy.post",
			Resource:  "fabric.FlowControlPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "ReceiveDirection", Validator: "disabled_string"}),
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "SendDirection", Validator: "disabled_string"}),
			},
		},
		{
			SDKMethod: "fabric.flowControlPolicy.update",
			Resource:  "fabric.FlowControlPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "ReceiveDirection", Validator: "disabled_string"}),
				contracts.NewConditionalInCustomRule("PriorityFlowControlMode", []any{"auto", "on"}, CustomRule{Field: "SendDirection", Validator: "disabled_string"}),
			},
		},
		{
			SDKMethod: "fabric.multicastPolicy.create",
			Resource:  "fabric.MulticastPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("QuerierState", "Enabled", FieldRule{Field: "QuerierIpAddress"}),
			},
		},
		{
			SDKMethod: "fabric.multicastPolicy.post",
			Resource:  "fabric.MulticastPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("QuerierState", "Enabled", FieldRule{Field: "QuerierIpAddress"}),
			},
		},
		{
			SDKMethod: "fabric.multicastPolicy.update",
			Resource:  "fabric.MulticastPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("QuerierState", "Enabled", FieldRule{Field: "QuerierIpAddress"}),
			},
		},
		{
			SDKMethod: "fabric.netFlowExporter.create",
			Resource:  "fabric.NetFlowExporter",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("NetFlowPolicy", "fabric.NetFlowPolicy"),
			},
		},
		{
			SDKMethod: "fabric.netFlowExporter.post",
			Resource:  "fabric.NetFlowExporter",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("NetFlowPolicy", "fabric.NetFlowPolicy"),
			},
		},
		{
			SDKMethod: "fabric.netFlowExporter.update",
			Resource:  "fabric.NetFlowExporter",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("NetFlowPolicy", "fabric.NetFlowPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcStorageRole.create",
			Resource:  "fabric.FcStorageRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcStorageRole.post",
			Resource:  "fabric.FcStorageRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcStorageRole.update",
			Resource:  "fabric.FcStorageRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkRole.create",
			Resource:  "fabric.FcUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkRole.post",
			Resource:  "fabric.FcUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkRole.update",
			Resource:  "fabric.FcUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkPcRole.create",
			Resource:  "fabric.FcUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkPcRole.post",
			Resource:  "fabric.FcUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcUplinkPcRole.update",
			Resource:  "fabric.FcUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkRole.create",
			Resource:  "fabric.FcoeUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkRole.post",
			Resource:  "fabric.FcoeUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkRole.update",
			Resource:  "fabric.FcoeUplinkRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkPcRole.create",
			Resource:  "fabric.FcoeUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkPcRole.post",
			Resource:  "fabric.FcoeUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.fcoeUplinkPcRole.update",
			Resource:  "fabric.FcoeUplinkPcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.applianceRole.create",
			Resource:  "fabric.ApplianceRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("EthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("EthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy"),
			},
		},
		{
			SDKMethod: "fabric.applianceRole.post",
			Resource:  "fabric.ApplianceRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("EthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("EthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy"),
			},
		},
		{
			SDKMethod: "fabric.applianceRole.update",
			Resource:  "fabric.ApplianceRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("EthNetworkControlPolicy", "fabric.EthNetworkControlPolicy"),
				contracts.NewRequiredRule("EthNetworkGroupPolicy", "fabric.EthNetworkGroupPolicy"),
			},
		},
		{
			SDKMethod: "fabric.appliancePcRole.create",
			Resource:  "fabric.AppliancePcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.appliancePcRole.post",
			Resource:  "fabric.AppliancePcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.appliancePcRole.update",
			Resource:  "fabric.AppliancePcRole",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
			},
		},
		{
			SDKMethod: "fabric.lanPinGroup.create",
			Resource:  "fabric.LanPinGroup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("PinTargetInterfaceRole", "fabric.AbstractInterfaceRole"),
			},
		},
		{
			SDKMethod: "fabric.lanPinGroup.post",
			Resource:  "fabric.LanPinGroup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("PinTargetInterfaceRole", "fabric.AbstractInterfaceRole"),
			},
		},
		{
			SDKMethod: "fabric.lanPinGroup.update",
			Resource:  "fabric.LanPinGroup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PortPolicy", "fabric.PortPolicy"),
				contracts.NewRequiredRule("PinTargetInterfaceRole", "fabric.AbstractInterfaceRole"),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.create",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.post",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.clusterReplicationNetworkPolicy.update",
			Resource:  "hyperflex.ClusterReplicationNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ReplicationIpranges", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.create",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.post",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.nodeConfigPolicy.update",
			Resource:  "hyperflex.NodeConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("MgmtIpRange", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.create",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpRootPwd", ""),
				contracts.NewRequiredRule("HypervisorAdmin", ""),
				contracts.NewRequiredRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.post",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpRootPwd", ""),
				contracts.NewRequiredRule("HypervisorAdmin", ""),
				contracts.NewRequiredRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.localCredentialPolicy.update",
			Resource:  "hyperflex.LocalCredentialPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpRootPwd", ""),
				contracts.NewRequiredRule("HypervisorAdmin", ""),
				contracts.NewRequiredRule("HypervisorAdminPwd", ""),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.create",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.post",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.proxySettingPolicy.update",
			Resource:  "hyperflex.ProxySettingPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewMinimumRule(MinimumRule{Field: "Port", Value: 1}),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.create",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpVersion", ""),
				contracts.NewRequiredRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.post",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpVersion", ""),
				contracts.NewRequiredRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.softwareVersionPolicy.update",
			Resource:  "hyperflex.SoftwareVersionPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("HxdpVersion", ""),
				contracts.NewRequiredRule("UpgradeTypes", "", 1),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.create",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.post",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.update",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.create",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.post",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.ucsmConfigPolicy.update",
			Resource:  "hyperflex.UcsmConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ServerFirmwareVersion", ""),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.create",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DnsServers", "", 1),
				contracts.NewRequiredRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.post",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DnsServers", "", 1),
				contracts.NewRequiredRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.sysConfigPolicy.update",
			Resource:  "hyperflex.SysConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DnsServers", "", 1),
				contracts.NewRequiredRule("NtpServers", "", 1),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.create",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DataCenter", ""),
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewRequiredRule("Username", ""),
				contracts.NewRequiredRule("Password", ""),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.post",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DataCenter", ""),
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewRequiredRule("Username", ""),
				contracts.NewRequiredRule("Password", ""),
			},
		},
		{
			SDKMethod: "hyperflex.vcenterConfigPolicy.update",
			Resource:  "hyperflex.VcenterConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("DataCenter", ""),
				contracts.NewRequiredRule("Hostname", ""),
				contracts.NewRequiredRule("Username", ""),
				contracts.NewRequiredRule("Password", ""),
			},
		},
		{
			SDKMethod: "ntp.policy.create",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.post",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.update",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "organization.organization.create",
			Resource:  "organization.Organization",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
			},
		},
		{
			SDKMethod: "organization.organization.post",
			Resource:  "organization.Organization",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
			},
		},
		{
			SDKMethod: "organization.organization.update",
			Resource:  "organization.Organization",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
			},
		},
		{
			SDKMethod: "fabric.portPolicy.create",
			Resource:  "fabric.PortPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "fabric.portPolicy.post",
			Resource:  "fabric.PortPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "fabric.portPolicy.update",
			Resource:  "fabric.PortPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "server.profile.create",
			Resource:  "server.Profile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "server.profile.post",
			Resource:  "server.Profile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "server.profile.update",
			Resource:  "server.Profile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Name", ""),
				contracts.NewRequiredRule("Organization", "organization.Organization"),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.create",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.post",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.backupConfigPolicy.update",
			Resource:  "recovery.BackupConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.create",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.post",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.update",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
			},
		},
		{
			SDKMethod: "recovery.onDemandBackup.create",
			Resource:  "recovery.OnDemandBackup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.onDemandBackup.post",
			Resource:  "recovery.OnDemandBackup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "recovery.onDemandBackup.update",
			Resource:  "recovery.OnDemandBackup",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("FileNamePrefix", ""),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.create",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.post",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.update",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policy.create",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("SenderEmail", ""),
				contracts.NewRequiredRule("SmtpPort", ""),
				contracts.NewRequiredRule("SmtpRecipients", "", 1),
				contracts.NewRequiredRule("SmtpServer", ""),
				contracts.NewRequiredRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "smtp.policy.post",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("SenderEmail", ""),
				contracts.NewRequiredRule("SmtpPort", ""),
				contracts.NewRequiredRule("SmtpRecipients", "", 1),
				contracts.NewRequiredRule("SmtpServer", ""),
				contracts.NewRequiredRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "smtp.policy.update",
			Resource:  "smtp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("SenderEmail", ""),
				contracts.NewRequiredRule("SmtpPort", ""),
				contracts.NewRequiredRule("SmtpRecipients", "", 1),
				contracts.NewRequiredRule("SmtpServer", ""),
				contracts.NewRequiredRule("MinSeverity", ""),
			},
		},
		{
			SDKMethod: "syslog.policy.create",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LocalClients", "syslog.LocalClientBase", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.post",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LocalClients", "syslog.LocalClientBase", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.update",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LocalClients", "syslog.LocalClientBase", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.create",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.post",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.update",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
			},
		},
		{
			SDKMethod: "server.diagnostics.create",
			Resource:  "server.Diagnostics",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ComponentList", "", 1),
			},
		},
		{
			SDKMethod: "server.diagnostics.post",
			Resource:  "server.Diagnostics",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ComponentList", "", 1),
			},
		},
		{
			SDKMethod: "server.diagnostics.update",
			Resource:  "server.Diagnostics",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ComponentList", "", 1),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.create",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.post",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "storage.driveSecurityPolicy.update",
			Resource:  "storage.DriveSecurityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("KeySetting", ""),
			},
		},
		{
			SDKMethod: "uuidpool.pool.create",
			Resource:  "uuidpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "uuidpool.pool.post",
			Resource:  "uuidpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "uuidpool.pool.update",
			Resource:  "uuidpool.Pool",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Prefix", ""),
			},
		},
		{
			SDKMethod: "uuidpool.reservation.create",
			Resource:  "uuidpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "uuidpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "uuidpool.reservation.post",
			Resource:  "uuidpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "uuidpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "uuidpool.reservation.update",
			Resource:  "uuidpool.Reservation",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("AllocationType", "Pool"),
				contracts.NewConditionalRequireRule("AllocationType", "dynamic", FieldRule{Field: "Pool", Target: "uuidpool.Pool"}),
				contracts.NewConditionalForbidRule("AllocationType", "static", "Pool"),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.create",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.post",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiAdapterPolicy.update",
			Resource:  "vnic.IscsiAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "DhcpTimeout", Value: 60}),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.create",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.post",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiBootPolicy.update",
			Resource:  "vnic.IscsiBootPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PrimaryTargetPolicy", ""),
			},
		},
		{
			SDKMethod: "vnic.fcAdapterPolicy.create",
			Resource:  "vnic.FcAdapterPolicy",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "ErrorDetectionTimeout", Value: 1000}),
			},
		},
		{
			SDKMethod: "vnic.fcQosPolicy.create",
			Resource:  "vnic.FcQosPolicy",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "MaxDataFieldSize", Value: 256}),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.create",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("IpAddress", ""),
				contracts.NewRequiredRule("IscsiIpType", ""),
				contracts.NewRequiredRule("Port", ""),
				contracts.NewRequiredRule("TargetName", ""),
				contracts.NewRequiredRule("Lun", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.post",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("IpAddress", ""),
				contracts.NewRequiredRule("IscsiIpType", ""),
				contracts.NewRequiredRule("Port", ""),
				contracts.NewRequiredRule("TargetName", ""),
				contracts.NewRequiredRule("Lun", ""),
			},
		},
		{
			SDKMethod: "vnic.iscsiStaticTargetPolicy.update",
			Resource:  "vnic.IscsiStaticTargetPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("IpAddress", ""),
				contracts.NewRequiredRule("IscsiIpType", ""),
				contracts.NewRequiredRule("Port", ""),
				contracts.NewRequiredRule("TargetName", ""),
				contracts.NewRequiredRule("Lun", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.create",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VlanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.post",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VlanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extIscsiStoragePolicy.update",
			Resource:  "hyperflex.ExtIscsiStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VlanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VlanId", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.create",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VsanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VsanId", ""),
				contracts.NewRequiredRule("WwxnPrefixRange", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.StartAddr", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.post",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VsanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VsanId", ""),
				contracts.NewRequiredRule("WwxnPrefixRange", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.StartAddr", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "hyperflex.extFcStoragePolicy.update",
			Resource:  "hyperflex.ExtFcStoragePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ExtaTraffic", ""),
				contracts.NewRequiredRule("ExtaTraffic.Name", ""),
				contracts.NewRequiredRule("ExtaTraffic.VsanId", ""),
				contracts.NewRequiredRule("ExtbTraffic", ""),
				contracts.NewRequiredRule("ExtbTraffic.Name", ""),
				contracts.NewRequiredRule("ExtbTraffic.VsanId", ""),
				contracts.NewRequiredRule("WwxnPrefixRange", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.StartAddr", ""),
				contracts.NewRequiredRule("WwxnPrefixRange.EndAddr", ""),
			},
		},
		{
			SDKMethod: "smtp.policyTest.create",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Policy", "smtp.Policy"),
				contracts.NewRequiredRule("Recipients", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policyTest.post",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Policy", "smtp.Policy"),
				contracts.NewRequiredRule("Recipients", "", 1),
			},
		},
		{
			SDKMethod: "smtp.policyTest.update",
			Resource:  "smtp.PolicyTest",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Policy", "smtp.Policy"),
				contracts.NewRequiredRule("Recipients", "", 1),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.create",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("StartDate", ""),
				contracts.NewOneOfRule("Entity", "AlarmRules"),
				contracts.NewEachRequiredRule("AlarmRules[].Property"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.post",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("StartDate", ""),
				contracts.NewOneOfRule("Entity", "AlarmRules"),
				contracts.NewEachRequiredRule("AlarmRules[].Property"),
			},
		},
		{
			SDKMethod: "cond.alarmSuppression.update",
			Resource:  "cond.AlarmSuppression",
			Rules: []SemanticRule{
				contracts.NewOneOfRule("Entity", "AlarmRules"),
				contracts.NewEachRequiredRule("AlarmRules[].Property"),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.create",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
				contracts.NewEachRequiredRule("PcieZones[].RootPcieEndpoint"),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.post",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
				contracts.NewEachRequiredRule("PcieZones[].RootPcieEndpoint"),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.update",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "compute.PcieZone", 1),
				contracts.NewEachRequiredRule("PcieZones[].RootPcieEndpoint"),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.create",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.post",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.update",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "vnic.ethNetworkPolicy.patch",
			Resource:  "vnic.EthNetworkPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "ACCESS", FieldRule{Field: "VlanSettings.DefaultVlan"}),
				contracts.NewConditionalRequireRule("VlanSettings.Mode", "TRUNK", FieldRule{Field: "VlanSettings.AllowedVlans"}),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.create",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.post",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.update",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.patch",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
				contracts.NewConditionalMinimumRule("VlanSettings.QinqEnabled", true, MinimumRule{Field: "VlanSettings.QinqVlan", Value: 2}),
			},
		},
		{
			SDKMethod: "asset.target.create",
			Resource:  "asset.Target",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Connections", "", 1),
			},
		},
		{
			SDKMethod: "asset.target.post",
			Resource:  "asset.Target",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Connections", "", 1),
			},
		},
		{
			SDKMethod: "asset.target.update",
			Resource:  "asset.Target",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Connections", "", 1),
			},
		},
		{
			SDKMethod: "firmware.serverConfigurationUtilityDistributable.create",
			Resource:  "firmware.ServerConfigurationUtilityDistributable",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("SupportedModels", "", 1),
			},
		},
		{
			SDKMethod: "firmware.serverConfigurationUtilityDistributable.post",
			Resource:  "firmware.ServerConfigurationUtilityDistributable",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("SupportedModels", "", 1),
			},
		},
		{
			SDKMethod: "firmware.serverConfigurationUtilityDistributable.update",
			Resource:  "firmware.ServerConfigurationUtilityDistributable",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("SupportedModels", "", 1),
			},
		},
		{
			SDKMethod: "workflow.ansibleBatchExecutor.create",
			Resource:  "workflow.AnsibleBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.ansibleBatchExecutor.post",
			Resource:  "workflow.AnsibleBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.ansibleBatchExecutor.update",
			Resource:  "workflow.AnsibleBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.batchApiExecutor.create",
			Resource:  "workflow.BatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.batchApiExecutor.post",
			Resource:  "workflow.BatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.batchApiExecutor.update",
			Resource:  "workflow.BatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.powerShellBatchApiExecutor.create",
			Resource:  "workflow.PowerShellBatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.powerShellBatchApiExecutor.post",
			Resource:  "workflow.PowerShellBatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.powerShellBatchApiExecutor.update",
			Resource:  "workflow.PowerShellBatchApiExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.sshBatchExecutor.create",
			Resource:  "workflow.SshBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.sshBatchExecutor.post",
			Resource:  "workflow.SshBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "workflow.sshBatchExecutor.update",
			Resource:  "workflow.SshBatchExecutor",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Batch", "", 1),
			},
		},
		{
			SDKMethod: "iam.appRegistration.create",
			Resource:  "iam.AppRegistration",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ClientName", ""),
			},
		},
		{
			SDKMethod: "iam.appRegistration.post",
			Resource:  "iam.AppRegistration",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ClientName", ""),
			},
		},
		{
			SDKMethod: "iam.appRegistration.update",
			Resource:  "iam.AppRegistration",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ClientName", ""),
			},
		},
		{
			SDKMethod: "iam.ldapProvider.create",
			Resource:  "iam.LdapProvider",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Server", ""),
			},
		},
		{
			SDKMethod: "iam.ldapProvider.post",
			Resource:  "iam.LdapProvider",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Server", ""),
			},
		},
		{
			SDKMethod: "iam.ldapProvider.update",
			Resource:  "iam.LdapProvider",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Server", ""),
			},
		},
		{
			SDKMethod: "mgmt.configBackupFile.create",
			Resource:  "mgmt.ConfigBackupFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Version", ""),
			},
		},
		{
			SDKMethod: "mgmt.configBackupFile.post",
			Resource:  "mgmt.ConfigBackupFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Version", ""),
			},
		},
		{
			SDKMethod: "mgmt.configBackupFile.update",
			Resource:  "mgmt.ConfigBackupFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Version", ""),
			},
		},
		{
			SDKMethod: "search.suggestItem.create",
			Resource:  "search.SuggestItem",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("SuggestTerm", ""),
			},
		},
		{
			SDKMethod: "softwarerepository.operatingSystemFile.create",
			Resource:  "softwarerepository.OperatingSystemFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Vendor", ""),
			},
		},
		{
			SDKMethod: "softwarerepository.operatingSystemFile.post",
			Resource:  "softwarerepository.OperatingSystemFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Vendor", ""),
			},
		},
		{
			SDKMethod: "softwarerepository.operatingSystemFile.update",
			Resource:  "softwarerepository.OperatingSystemFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Vendor", ""),
			},
		},
		{
			SDKMethod: "workflow.catalogItemDefinition.create",
			Resource:  "workflow.CatalogItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.catalogItemDefinition.post",
			Resource:  "workflow.CatalogItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.catalogItemDefinition.update",
			Resource:  "workflow.CatalogItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.customDataTypeDefinition.create",
			Resource:  "workflow.CustomDataTypeDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.customDataTypeDefinition.post",
			Resource:  "workflow.CustomDataTypeDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.customDataTypeDefinition.update",
			Resource:  "workflow.CustomDataTypeDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemActionDefinition.create",
			Resource:  "workflow.ServiceItemActionDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemActionDefinition.post",
			Resource:  "workflow.ServiceItemActionDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemActionDefinition.update",
			Resource:  "workflow.ServiceItemActionDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemDefinition.create",
			Resource:  "workflow.ServiceItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemDefinition.post",
			Resource:  "workflow.ServiceItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.serviceItemDefinition.update",
			Resource:  "workflow.ServiceItemDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.taskDefinition.create",
			Resource:  "workflow.TaskDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.taskDefinition.post",
			Resource:  "workflow.TaskDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.taskDefinition.update",
			Resource:  "workflow.TaskDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.workflowDefinition.create",
			Resource:  "workflow.WorkflowDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.workflowDefinition.post",
			Resource:  "workflow.WorkflowDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.workflowDefinition.update",
			Resource:  "workflow.WorkflowDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workflow.workflowInfo.create",
			Resource:  "workflow.WorkflowInfo",
			Rules: []SemanticRule{
				contracts.NewMinimumRule(MinimumRule{Field: "FailedWorkflowCleanupDuration", Value: 1}),
				contracts.NewMinimumRule(MinimumRule{Field: "SuccessWorkflowCleanupDuration", Value: 1}),
				contracts.NewConditionalForbidRule("Action", "None", "Action"),
			},
		},
		{
			SDKMethod: "workload.blueprint.create",
			Resource:  "workload.Blueprint",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
				contracts.NewRequiredRule("ServiceItems", "", 1),
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9_]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.blueprint.post",
			Resource:  "workload.Blueprint",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "workload.blueprint.update",
			Resource:  "workload.Blueprint",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Label", ""),
			},
		},
		{
			SDKMethod: "os.templateFile.create",
			Resource:  "os.TemplateFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("TemplateContent", ""),
				contracts.NewPatternRule(PatternRule{Field: "TemplateContent", Value: ".*\\S.*"}),
			},
		},
		{
			SDKMethod: "os.templateFile.post",
			Resource:  "os.TemplateFile",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("TemplateContent", ""),
				contracts.NewPatternRule(PatternRule{Field: "TemplateContent", Value: ".*\\S.*"}),
			},
		},
		{
			SDKMethod: "workflow.templateParser.create",
			Resource:  "workflow.TemplateParser",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("TemplateContent", ""),
				contracts.NewPatternRule(PatternRule{Field: "TemplateContent", Value: ".*\\S.*"}),
			},
		},
		{
			SDKMethod: "workflow.templateParser.post",
			Resource:  "workflow.TemplateParser",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("TemplateContent", ""),
				contracts.NewPatternRule(PatternRule{Field: "TemplateContent", Value: ".*\\S.*"}),
			},
		},
		{
			SDKMethod: "iam.endPointUser.create",
			Resource:  "iam.EndPointUser",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "iam.endPointUser.post",
			Resource:  "iam.EndPointUser",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "iam.endPointUser.update",
			Resource:  "iam.EndPointUser",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vhbaTemplate.create",
			Resource:  "vnic.VhbaTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vhbaTemplate.post",
			Resource:  "vnic.VhbaTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vhbaTemplate.update",
			Resource:  "vnic.VhbaTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vnicTemplate.create",
			Resource:  "vnic.VnicTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vnicTemplate.post",
			Resource:  "vnic.VnicTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "vnic.vnicTemplate.update",
			Resource:  "vnic.VnicTemplate",
			Rules: []SemanticRule{
				contracts.NewMaximumRule(LengthRule{Field: "Name", Value: 16}),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.create",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
				contracts.NewFutureRule("Schedule.ExecutionTime"),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.post",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
				contracts.NewFutureRule("Schedule.ExecutionTime"),
			},
		},
		{
			SDKMethod: "recovery.scheduleConfigPolicy.update",
			Resource:  "recovery.ScheduleConfigPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Schedule", ""),
				contracts.NewRequiredRule("Schedule.ExecutionTime", ""),
				contracts.NewRequiredRule("Schedule.FrequencyUnit", ""),
				contracts.NewFutureRule("Schedule.ExecutionTime"),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.create",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
				contracts.NewFutureRule("ScheduleParams[].StartTime"),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.post",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
				contracts.NewFutureRule("ScheduleParams[].StartTime"),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.update",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "scheduler.BaseScheduleParams", 1),
				contracts.NewFutureRule("ScheduleParams[].StartTime"),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.create",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
				contracts.NewCustomRule(CustomRule{Field: "BaseProperties.Filter", Validator: "ldap_filter"}),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.post",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
				contracts.NewCustomRule(CustomRule{Field: "BaseProperties.Filter", Validator: "ldap_filter"}),
			},
		},
		{
			SDKMethod: "iam.ldapPolicy.update",
			Resource:  "iam.LdapPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("BaseProperties", ""),
				contracts.NewCustomRule(CustomRule{Field: "BaseProperties.Filter", Validator: "ldap_filter"}),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.create",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
				contracts.NewContainsRule(ContainsRule{Field: "Qualifiers[].ObjectType", Value: "resource.GpuQualifier"}),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.post",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
				contracts.NewContainsRule(ContainsRule{Field: "Qualifiers[].ObjectType", Value: "resource.GpuQualifier"}),
			},
		},
		{
			SDKMethod: "resourcepool.qualificationPolicy.update",
			Resource:  "resourcepool.QualificationPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Qualifiers", "", 1),
				contracts.NewContainsRule(ContainsRule{Field: "Qualifiers[].ObjectType", Value: "resource.GpuQualifier"}),
			},
		},
		{
			SDKMethod: "workload.workloadDefinition.create",
			Resource:  "workload.WorkloadDefinition",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Blueprints", "", 1),
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDefinition.post",
			Resource:  "workload.WorkloadDefinition",
			Rules: []SemanticRule{
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDefinition.update",
			Resource:  "workload.WorkloadDefinition",
			Rules: []SemanticRule{
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDeployment.create",
			Resource:  "workload.WorkloadDeployment",
			Rules: []SemanticRule{
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDeployment.post",
			Resource:  "workload.WorkloadDeployment",
			Rules: []SemanticRule{
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
		{
			SDKMethod: "workload.workloadDeployment.update",
			Resource:  "workload.WorkloadDeployment",
			Rules: []SemanticRule{
				contracts.NewPatternRule(PatternRule{Field: "Name", Value: "^[a-zA-Z0-9][a-zA-Z0-9- _]{0,31}$"}),
			},
		},
	}
}
