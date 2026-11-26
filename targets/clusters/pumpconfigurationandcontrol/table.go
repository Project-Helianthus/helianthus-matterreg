// Package pumpconfigurationandcontrol is the Pump Configuration and Control cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package pumpconfigurationandcontrol

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0200, Name: "Pump Configuration and Control", Revision: 5,
		Features: []clusters.Feature{
			{Bit: 0, Code: "PRSCONST", Name: "ConstantPressure", Conformance: "O.a+"},
			{Bit: 1, Code: "PRSCOMP", Name: "CompensatedPressure", Conformance: "O.a+"},
			{Bit: 2, Code: "FLW", Name: "ConstantFlow", Conformance: "O.a+"},
			{Bit: 3, Code: "SPD", Name: "ConstantSpeed", Conformance: "O.a+"},
			{Bit: 4, Code: "TEMP", Name: "ConstantTemperature", Conformance: "O.a+"},
			{Bit: 5, Code: "AUTO", Name: "Automatic", Conformance: "O"},
			{Bit: 6, Code: "LOCAL", Name: "LocalOperation", Conformance: "O"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "MaxPressure", MatterType: "int16", Nullable: true, Conformance: "M", Access: "R V", Quality: "X F", Mandatory: true},
			{ID: 0x0001, Name: "MaxSpeed", MatterType: "uint16", Nullable: true, Conformance: "M", Access: "R V", Quality: "X F", Mandatory: true},
			{ID: 0x0002, Name: "MaxFlow", MatterType: "uint16", Nullable: true, Conformance: "M", Access: "R V", Quality: "X F", Mandatory: true},
			{ID: 0x0003, Name: "MinConstPressure", MatterType: "int16", Nullable: true, Conformance: "PRSCONST, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0004, Name: "MaxConstPressure", MatterType: "int16", Nullable: true, Conformance: "PRSCONST, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0005, Name: "MinCompPressure", MatterType: "int16", Nullable: true, Conformance: "PRSCOMP, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0006, Name: "MaxCompPressure", MatterType: "int16", Nullable: true, Conformance: "PRSCOMP, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0007, Name: "MinConstSpeed", MatterType: "uint16", Nullable: true, Conformance: "SPD, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0008, Name: "MaxConstSpeed", MatterType: "uint16", Nullable: true, Conformance: "SPD, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0009, Name: "MinConstFlow", MatterType: "uint16", Nullable: true, Conformance: "FLW, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x000A, Name: "MaxConstFlow", MatterType: "uint16", Nullable: true, Conformance: "FLW, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x000B, Name: "MinConstTemp", MatterType: "int16", Nullable: true, Conformance: "TEMP, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x000C, Name: "MaxConstTemp", MatterType: "int16", Nullable: true, Conformance: "TEMP, [AUTO]", Access: "R V", Quality: "X F", Mandatory: false},
			{ID: 0x0010, Name: "PumpStatus", MatterType: "PumpStatusBitmap", Nullable: false, Conformance: "O", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0011, Name: "EffectiveOperationMode", MatterType: "OperationModeEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0012, Name: "EffectiveControlMode", MatterType: "ControlModeEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0013, Name: "Capacity", MatterType: "int16", Nullable: true, Conformance: "M", Access: "R V", Quality: "X", Mandatory: true},
			{ID: 0x0014, Name: "Speed", MatterType: "uint16", Nullable: true, Conformance: "O", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0015, Name: "LifetimeRunningHours", MatterType: "uint24", Nullable: true, Conformance: "O", Access: "RW VM", Quality: "X N", Mandatory: false},
			{ID: 0x0016, Name: "Power", MatterType: "uint24", Nullable: true, Conformance: "O", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0x0017, Name: "LifetimeEnergyConsumed", MatterType: "uint32", Nullable: true, Conformance: "O", Access: "RW VM", Quality: "X N", Mandatory: false},
			{ID: 0x0020, Name: "OperationMode", MatterType: "OperationModeEnum", Nullable: false, Conformance: "M", Access: "RW VM", Quality: "N", Mandatory: true},
			{ID: 0x0021, Name: "ControlMode", MatterType: "ControlModeEnum", Nullable: false, Conformance: "O", Access: "RW VM", Quality: "N", Mandatory: false},
			{ID: 0x0022, Name: "AlarmMask", MatterType: "uint16", Nullable: false, Conformance: "D", Access: "RW VO", Quality: "", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
	}
}
