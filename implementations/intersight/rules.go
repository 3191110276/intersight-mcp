package intersight

import "github.com/mimaurer/intersight-mcp/internal/contracts"

type RuleTemplate = contracts.RuleTemplate
type SemanticRule = contracts.SemanticRule
type FieldRule = contracts.FieldRule
type MinimumRule = contracts.MinimumRule

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
			SDKMethod: "compute.pcieConnectivityPolicy.create",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.post",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "", 1),
			},
		},
		{
			SDKMethod: "compute.pcieConnectivityPolicy.update",
			Resource:  "compute.PcieConnectivityPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("PcieZones", "", 1),
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
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.post",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
			},
		},
		{
			SDKMethod: "fabric.ethNetworkGroupPolicy.update",
			Resource:  "fabric.EthNetworkGroupPolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("VlanSettings", ""),
				contracts.NewRequiredRule("VlanSettings.AllowedVlans", ""),
				contracts.NewConditionalRequireRule("VlanSettings.QinqEnabled", true, FieldRule{Field: "VlanSettings.QinqVlan"}),
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
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.post",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
			},
		},
		{
			SDKMethod: "ntp.policy.update",
			Resource:  "ntp.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("Enabled", ""),
				contracts.NewRequiredRule("Timezone", ""),
				contracts.NewOneOfRule("NtpServers", "AuthenticatedNtpServers"),
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
				contracts.NewRequiredRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.post",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "syslog.policy.update",
			Resource:  "syslog.Policy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("LocalClients", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.create",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.post",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "", 1),
			},
		},
		{
			SDKMethod: "scheduler.schedulePolicy.update",
			Resource:  "scheduler.SchedulePolicy",
			Rules: []SemanticRule{
				contracts.NewRequiredRule("ScheduleParams", "", 1),
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
	}
}
