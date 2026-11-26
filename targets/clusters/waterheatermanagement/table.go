// Package waterheatermanagement is the Water Heater Management cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package waterheatermanagement

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0094, Name: "Water Heater Management", Revision: 2,
		Features: []clusters.Feature{
			{Bit: 0, Code: "EM", Name: "EnergyManagement", Conformance: "O"},
			{Bit: 1, Code: "TP", Name: "TankPercent", Conformance: "O"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "HeaterTypes", MatterType: "WaterHeaterHeatSourceBitmap", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0001, Name: "HeatDemand", MatterType: "WaterHeaterHeatSourceBitmap", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0x0002, Name: "TankVolume", MatterType: "uint16", Nullable: false, Conformance: "EM", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0003, Name: "EstimatedHeatRequired", MatterType: "energy-mWh", Nullable: false, Conformance: "EM", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0004, Name: "TankPercentage", MatterType: "percent", Nullable: false, Conformance: "TP", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x0005, Name: "BoostState", MatterType: "BoostStateEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "Boost", Direction: "request → status", Conformance: "M"},
			{ID: 0x01, Name: "CancelBoost", Direction: "request → status", Conformance: "M"},
		},
	}
}
