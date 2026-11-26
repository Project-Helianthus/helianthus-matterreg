// Package electricalpowermeasurement is the Electrical Power Measurement cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package electricalpowermeasurement

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0090, Name: "Electrical Power Measurement", Revision: 3,
		Features: []clusters.Feature{
			{Bit: 0, Code: "DIRC", Name: "DirectCurrent", Conformance: "O.a+"},
			{Bit: 1, Code: "ALTC", Name: "AlternatingCurrent", Conformance: "O.a+"},
			{Bit: 2, Code: "POLY", Name: "PolyphasePower", Conformance: "[ALTC]"},
			{Bit: 3, Code: "HARM", Name: "Harmonics", Conformance: "[ALTC]"},
			{Bit: 4, Code: "PWRQ", Name: "PowerQuality", Conformance: "[ALTC]"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "PowerMode", MatterType: "PowerModeEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0001, Name: "NumberOfMeasurementTypes", MatterType: "uint8", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0002, Name: "Accuracy", MatterType: "list[MeasurementAccuracyStruct]", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0003, Name: "Ranges", MatterType: "list[MeasurementRangeStruct]", Nullable: false, Conformance: "O", Access: "R V", Quality: "Q", Mandatory: false},
			{ID: 0x0004, Name: "Voltage", MatterType: "voltage-mV", Nullable: true, Conformance: "O", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0005, Name: "ActiveCurrent", MatterType: "amperage-mA", Nullable: true, Conformance: "O", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0006, Name: "ReactiveCurrent", MatterType: "amperage-mA", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0007, Name: "ApparentCurrent", MatterType: "amperage-mA", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0008, Name: "ActivePower", MatterType: "power-mW", Nullable: true, Conformance: "M", Access: "R V", Quality: "X Q", Mandatory: true},
			{ID: 0x0009, Name: "ReactivePower", MatterType: "power-mVAR", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000A, Name: "ApparentPower", MatterType: "power-mVA", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000B, Name: "RmsVoltage", MatterType: "voltage-mV", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000C, Name: "RmsCurrent", MatterType: "amperage-mA", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000D, Name: "RmsPower", MatterType: "power-mW", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000E, Name: "Frequency", MatterType: "int64", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x000F, Name: "HarmonicCurrents", MatterType: "list[HarmonicMeasurementStruct]", Nullable: true, Conformance: "HARM", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0010, Name: "HarmonicPhases", MatterType: "list[HarmonicMeasurementStruct]", Nullable: true, Conformance: "PWRQ", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0011, Name: "PowerFactor", MatterType: "int64", Nullable: true, Conformance: "[ALTC]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0012, Name: "NeutralCurrent", MatterType: "amperage-mA", Nullable: true, Conformance: "[POLY]", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Mappings: []clusters.Mapping{{
			AttributeID: 0x0005, PackID: "helianthus.pack.evse", PackVersion: "1.0.0", Element: "evse.ac.current",
			Dimension: "evse.dimension.phase", Encoder: "ampere_to_milliampere", Reason: "matter.phase_association_by_relation.v1",
			FromUnit: "unit.ampere", ToUnit: "amperage-mA", Factor: 1000, Offset: 0, Rounding: "exact", RequireIntegral: true,
			Minimum: "-4611686018427387904", Maximum: "4611686018427387904",
		}},
	}
}
