// Package powertopology is the Power Topology cluster table of Matter 1.6.1: identity, feature map,
// attributes and commands with their conformance, and the transformed rows the target maps.
package powertopology

import "github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x009C, Name: "Power Topology", Revision: 1,
		Features: []clusters.Feature{
			{Bit: 0, Code: "NODE", Name: "NodeTopology", Conformance: "O.a"},
			{Bit: 1, Code: "TREE", Name: "TreeTopology", Conformance: "O.a"},
			{Bit: 2, Code: "SET", Name: "SetTopology", Conformance: "O.a"},
			{Bit: 3, Code: "DYPF", Name: "DynamicPowerFlow", Conformance: "[SET]"},
		},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "AvailableEndpoints", MatterType: "list[endpoint-no]", Nullable: false, Conformance: "SET", Access: "R V", Quality: "F", Mandatory: false},
			{ID: 0x0001, Name: "ActiveEndpoints", MatterType: "list[endpoint-no]", Nullable: false, Conformance: "DYPF", Access: "R V", Quality: "N", Mandatory: false},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
	}
}
