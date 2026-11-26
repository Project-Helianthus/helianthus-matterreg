// Package identify is the Identify cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance. The target maps no attribute of this cluster, so the table carries
// no transformed row, no encoder and no operation.
package identify

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0003, Name: "Identify", Revision: 6,
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "IdentifyTime", MatterType: "uint16", Nullable: false, Conformance: "M", Access: "RW VO", Quality: "Q", Mandatory: true},
			{ID: 0x0001, Name: "IdentifyType", MatterType: "IdentifyTypeEnum", Nullable: false, Conformance: "M", Access: "R V", Quality: "", Mandatory: true},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{
			{ID: 0x00, Name: "Identify", Direction: "request → status", Conformance: "M"},
			{ID: 0x40, Name: "TriggerEffect", Direction: "request → status", Conformance: "O"},
		},
	}
}
