// Package energyevsemode is the Energy EVSE Mode cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package energyevsemode

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x009D, Name: "Energy EVSE Mode", Revision: 2,
		Features: []clusters.Feature{
			{Bit: 0, Code: "DEPONOFF", Name: "OnOff", Conformance: "X"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "SupportedModes", MatterType: "list[ModeOptionStruct]", Nullable: false, Conformance: "M", Access: "R V", Quality: "F", Mandatory: true},
			{ID: 0x0001, Name: "CurrentMode", MatterType: "uint8", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0x0002, Name: "StartUpMode", MatterType: "uint8", Nullable: true, Conformance: "X", Access: "RW VO", Quality: "X N", Mandatory: false},
			{ID: 0x0003, Name: "OnMode", MatterType: "uint8", Nullable: true, Conformance: "X", Access: "RW VO", Quality: "X N", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "ChangeToMode", Direction: "request → ChangeToModeResponse", Conformance: "M"},
			{ID: 0x01, Name: "ChangeToModeResponse", Direction: "response", Conformance: "M"},
		},
	}
}
