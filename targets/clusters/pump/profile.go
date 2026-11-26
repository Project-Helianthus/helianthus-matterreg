// Package pump is the Pump profile of Matter 1.6.1 for the pump resource
// kind. The device type requires the Identify, On/Off and Pump Configuration
// and Control servers on the root and no nested device type. The target maps
// no transformed row for this kind, so every dynamic mandatory element is
// missing with the reason named and nothing is invented. The control features
// of Pump Configuration and Control are an open choice the device type does not
// settle, so its FeatureMap is missing too.
package pump

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/identify"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/onoff"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/pumpconfigurationandcontrol"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identity.
const (
	DeviceTypeID       uint32 = 0x0303
	DeviceTypeRevision uint32 = 4
)

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.pump"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0004, Name: "Groups", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0006, Name: "On/Off", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0008, Name: "Level Control", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0062, Name: "Scenes Management", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0200, Name: "Pump Configuration and Control", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0402, Name: "Temperature Measurement", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0403, Name: "Pressure Measurement", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0404, Name: "Flow Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0402, Name: "Temperature Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0403, Name: "Pressure Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0404, Name: "Flow Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0406, Name: "Occupancy Sensing", Conformance: "O"},
	}
}

func identifyCluster() partrows.Cluster {
	return partrows.Cluster{Table: identify.Table()}
}

func onOff() partrows.Cluster {
	return partrows.Cluster{Table: onoff.Table()}
}

func configuration() partrows.Cluster {
	return partrows.Cluster{
		Table:         pumpconfigurationandcontrol.Table(),
		FeatureChoice: "at least one of PRSCONST, PRSCOMP, FLW, SPD and TEMP is required and the device type fixes none",
	}
}

// Parts returns the required parts; the device type requires none.
func Parts() []composition.PartSpec {
	return []composition.PartSpec{}
}

// Servers returns the server clusters mounted on the root endpoint besides
// Descriptor.
func Servers() []composition.ServerRow {
	return partrows.Servers(identifyCluster().Table, onOff().Table, configuration().Table)
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes and the rows of the three servers.
func Rows() []composition.AttributeRow {
	id, oo, pc := identifyCluster(), onOff(), configuration()
	rows := partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision,
		[]uint32{0x001D, id.Table.ClusterID, oo.Table.ClusterID, pc.Table.ClusterID}, false, nil)
	rows = append(rows, id.Rows("")...)
	rows = append(rows, oo.Rows("")...)
	rows = append(rows, pc.Rows("")...)
	return rows
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	return partrows.Missing(Rows(), Parts())
}

// Folds returns the declared losses. The target names no fold for this kind.
func Folds() []composition.Fold {
	return []composition.Fold{}
}
