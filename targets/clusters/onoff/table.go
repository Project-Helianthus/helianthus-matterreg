// Package onoff is the On/Off cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package onoff

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0006, Name: "On/Off", Revision: 6,
		Features: []clusters.Feature{
			{Bit: 0, Code: "LT", Name: "Lighting", Conformance: "[!OFFONLY]"},
			{Bit: 1, Code: "DF", Name: "DeadFrontBehavior", Conformance: "[!OFFONLY]"},
			{Bit: 2, Code: "OFFONLY", Name: "OffOnly", Conformance: "[!(LT | DF)]"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "OnOff", MatterType: "bool", Nullable: false, Conformance: "M", Access: "R V", Quality: "N S", Mandatory: true},
			{ID: 0x4000, Name: "GlobalSceneControl", MatterType: "bool", Nullable: false, Conformance: "LT", Access: "R V", Quality: "", Mandatory: false},
			{ID: 0x4001, Name: "OnTime", MatterType: "uint16", Nullable: false, Conformance: "LT", Access: "RW VO", Quality: "", Mandatory: false},
			{ID: 0x4002, Name: "OffWaitTime", MatterType: "uint16", Nullable: false, Conformance: "LT", Access: "RW VO", Quality: "", Mandatory: false},
			{ID: 0x4003, Name: "StartUpOnOff", MatterType: "StartUpOnOffEnum", Nullable: true, Conformance: "LT", Access: "RW VM", Quality: "X N", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "Off", Direction: "request → status", Conformance: "M"},
			{ID: 0x01, Name: "On", Direction: "request → status", Conformance: "!OFFONLY"},
			{ID: 0x02, Name: "Toggle", Direction: "request → status", Conformance: "!OFFONLY"},
			{ID: 0x40, Name: "OffWithEffect", Direction: "request → status", Conformance: "LT"},
			{ID: 0x41, Name: "OnWithRecallGlobalScene", Direction: "request → status", Conformance: "LT"},
			{ID: 0x42, Name: "OnWithTimedOff", Direction: "request → status", Conformance: "LT"},
		},
	}
}
