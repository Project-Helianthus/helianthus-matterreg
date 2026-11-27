// Package electricalpowermeasurement is the Electrical Power Measurement cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package electricalpowermeasurement

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Percent100thsPerRatio is the number of percent100ths in a ratio of one, and
// Percent100thsMaximum is the maximum the percent100ths type states.
const (
	Percent100thsPerRatio = 10000
	Percent100thsMaximum  = 10000
)

// ElectricalEnergyMeasurementClusterID is the cluster whose single Accuracy
// struct carries the energy measurement type.
const ElectricalEnergyMeasurementClusterID uint32 = 0x0091

// MeasurementType is one member of MeasurementTypeEnum. Unit is the SemReg
// unit of the type's measured and range figures, empty where the catalog has no
// such element; ClusterID is the cluster whose Accuracy attribute carries the
// type, zero where no Accuracy attribute carries it.
type MeasurementType struct {
	Symbol     string
	ID         uint16
	Name       string
	ClusterID  uint32
	SemregUnit string
}

// MeasurementTypes returns a copy of the MeasurementTypeEnum rows.
func MeasurementTypes() []MeasurementType {
	return []MeasurementType{
		{"unspecified", 0, "Unspecified", 0x0090, ""},
		{"voltage", 1, "Voltage", 0x0090, ""},
		{"active_current", 2, "ActiveCurrent", 0x0090, "unit.ampere"},
		{"reactive_current", 3, "ReactiveCurrent", 0x0090, ""},
		{"apparent_current", 4, "ApparentCurrent", 0x0090, ""},
		{"active_power", 5, "ActivePower", 0x0090, "unit.watt"},
		{"reactive_power", 6, "ReactivePower", 0x0090, ""},
		{"apparent_power", 7, "ApparentPower", 0x0090, ""},
		{"rms_voltage", 8, "RmsVoltage", 0x0090, ""},
		{"rms_current", 9, "RmsCurrent", 0x0090, ""},
		{"rms_power", 10, "RmsPower", 0x0090, ""},
		{"frequency", 11, "Frequency", 0x0090, ""},
		{"power_factor", 12, "PowerFactor", 0x0090, ""},
		{"neutral_current", 13, "NeutralCurrent", 0x0090, ""},
		{"electrical_energy", 14, "ElectricalEnergy", ElectricalEnergyMeasurementClusterID, ""},
		{"reactive_energy", 15, "ReactiveEnergy", 0, ""},
		{"apparent_energy", 16, "ApparentEnergy", 0, ""},
	}
}

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
