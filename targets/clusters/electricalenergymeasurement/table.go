// Package electricalenergymeasurement is the Electrical Energy Measurement cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package electricalenergymeasurement

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0091, Name: "Electrical Energy Measurement", Revision: 2,
		Features: []clusters.Feature{
			{Bit: 0, Code: "IMPE", Name: "ImportedEnergy", Conformance: "O.a+"},
			{Bit: 1, Code: "EXPE", Name: "ExportedEnergy", Conformance: "O.a+"},
			{Bit: 2, Code: "CUME", Name: "CumulativeEnergy", Conformance: "O.b+"},
			{Bit: 3, Code: "PERE", Name: "PeriodicEnergy", Conformance: "O.b+"},
			{Bit: 4, Code: "APPE", Name: "ApparentEnergy", Conformance: "P, O"},
			{Bit: 5, Code: "REAE", Name: "ReactiveEnergy", Conformance: "P, O"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "Accuracy", MatterType: "MeasurementAccuracyStruct", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0001, Name: "CumulativeEnergyImported", MatterType: "EnergyMeasurementStruct", Nullable: true, Conformance: "IMPE & CUME", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0002, Name: "CumulativeEnergyExported", MatterType: "EnergyMeasurementStruct", Nullable: true, Conformance: "EXPE & CUME", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0003, Name: "PeriodicEnergyImported", MatterType: "EnergyMeasurementStruct", Nullable: true, Conformance: "IMPE & PERE", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0004, Name: "PeriodicEnergyExported", MatterType: "EnergyMeasurementStruct", Nullable: true, Conformance: "EXPE & PERE", Access: "R V", Quality: "X Q", Mandatory: false},
			{ID: 0x0005, Name: "CumulativeEnergyReset", MatterType: "CumulativeEnergyResetStruct", Nullable: true, Conformance: "[CUME]", Access: "R V", Quality: "X", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
	}
}
