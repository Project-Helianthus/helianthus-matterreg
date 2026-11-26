// Package powersource is the Power Source cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package powersource

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x002F, Name: "Power Source", Revision: 3,
		Features: []clusters.Feature{
			{Bit: 0, Code: "WIRED", Name: "Wired", Conformance: "O.a"},
			{Bit: 1, Code: "BAT", Name: "Battery", Conformance: "O.a"},
			{Bit: 2, Code: "RECHG", Name: "Rechargeable", Conformance: "[BAT]"},
			{Bit: 3, Code: "REPLC", Name: "Replaceable", Conformance: "[BAT]"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "Status", MatterType: "PowerSourceStatusEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0001, Name: "Order", MatterType: "uint8", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0002, Name: "Description", MatterType: "string", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0003, Name: "WiredAssessedInputVoltage", MatterType: "uint32", Nullable: true, Conformance: "[WIRED]", Access: "R V", Quality: "X C", Mandatory: false},
			{ID: 0x0004, Name: "WiredAssessedInputFrequency", MatterType: "uint16", Nullable: true, Conformance: "[WIRED]", Access: "R V", Quality: "X C", Mandatory: false},
			{ID: 0x0005, Name: "WiredCurrentType", MatterType: "WiredCurrentTypeEnum", Nullable: false, Conformance: "WIRED", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0006, Name: "WiredAssessedCurrent", MatterType: "uint32", Nullable: true, Conformance: "[WIRED]", Access: "R V", Quality: "X C", Mandatory: false},
			{ID: 0x0007, Name: "WiredNominalVoltage", MatterType: "uint32", Nullable: false, Conformance: "[WIRED]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0008, Name: "WiredMaximumCurrent", MatterType: "uint32", Nullable: false, Conformance: "[WIRED]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0009, Name: "WiredPresent", MatterType: "bool", Nullable: false, Conformance: "[WIRED]", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x000A, Name: "ActiveWiredFaults", MatterType: "list[WiredFaultEnum]", Nullable: false, Conformance: "[WIRED]", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x000B, Name: "BatVoltage", MatterType: "uint32", Nullable: true, Conformance: "[BAT]", Access: "R V", Quality: "X C", Mandatory: false},
			{ID: 0x000C, Name: "BatPercentRemaining", MatterType: "uint8", Nullable: true, Conformance: "[BAT]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000D, Name: "BatTimeRemaining", MatterType: "uint32", Nullable: true, Conformance: "[BAT]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000E, Name: "BatChargeLevel", MatterType: "BatChargeLevelEnum", Nullable: false, Conformance: "BAT", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x000F, Name: "BatReplacementNeeded", MatterType: "bool", Nullable: false, Conformance: "BAT", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0010, Name: "BatReplaceability", MatterType: "BatReplaceabilityEnum", Nullable: false, Conformance: "BAT", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0011, Name: "BatPresent", MatterType: "bool", Nullable: false, Conformance: "[BAT]", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0012, Name: "ActiveBatFaults", MatterType: "list[BatFaultEnum]", Nullable: false, Conformance: "[BAT]", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0013, Name: "BatReplacementDescription", MatterType: "string", Nullable: false, Conformance: "REPLC", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0014, Name: "BatCommonDesignation", MatterType: "BatCommonDesignationEnum", Nullable: false, Conformance: "[REPLC]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0015, Name: "BatAnsiDesignation", MatterType: "string", Nullable: false, Conformance: "[REPLC]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0016, Name: "BatIecDesignation", MatterType: "string", Nullable: false, Conformance: "[REPLC]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0017, Name: "BatApprovedChemistry", MatterType: "BatApprovedChemistryEnum", Nullable: false, Conformance: "[REPLC]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0018, Name: "BatCapacity", MatterType: "uint32", Nullable: false, Conformance: "[REPLC | RECHG]", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0019, Name: "BatQuantity", MatterType: "uint8", Nullable: false, Conformance: "REPLC", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x001A, Name: "BatChargeState", MatterType: "BatChargeStateEnum", Nullable: false, Conformance: "RECHG", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x001B, Name: "BatTimeToFullCharge", MatterType: "uint32", Nullable: true, Conformance: "[RECHG]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x001C, Name: "BatFunctionalWhileCharging", MatterType: "bool", Nullable: false, Conformance: "RECHG", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x001D, Name: "BatChargingCurrent", MatterType: "uint32", Nullable: true, Conformance: "[RECHG]", Access: "R V", Quality: "X C", Mandatory: false},
			{ID: 0x001E, Name: "ActiveBatChargeFaults", MatterType: "list[BatChargeFaultEnum]", Nullable: false, Conformance: "[RECHG]", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x001F, Name: "EndpointList", MatterType: "list[endpoint-no]", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
	}
}
