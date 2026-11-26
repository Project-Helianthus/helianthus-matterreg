// Package zone is the Thermostat profile of Matter 1.6.1 for the zone
// resource kind. The device type requires the Identify and Thermostat servers
// on the root and no nested device type. The target maps no transformed row
// for this kind, so every dynamic mandatory element is missing with the reason
// named and nothing is invented. The Thermostat features are an open choice
// the device type does not settle, so the Thermostat FeatureMap is missing too.
package zone

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/identify"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/thermostat"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identity.
const (
	DeviceTypeID       uint32 = 0x0301
	DeviceTypeRevision uint32 = 7
)

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.zone"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0004, Name: "Groups", Conformance: "Active", Note: "conditional on the node's Active condition; nothing establishes it here; not mounted"},
		{Element: "server_cluster", ID: 0x009B, Name: "Energy Preference", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0201, Name: "Thermostat", Conformance: "M", Mounted: true},
		{Element: "client_cluster", ID: 0x0202, Name: "Fan Control", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0204, Name: "Thermostat User Interface Configuration", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0402, Name: "Temperature Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0405, Name: "Relative Humidity Measurement", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0406, Name: "Occupancy Sensing", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0431, Name: "Ambient Context Sensing", Conformance: "P, [Rev >= v6]"},
	}
}

func identifyCluster() partrows.Cluster {
	return partrows.Cluster{Table: identify.Table()}
}

func thermostatCluster() partrows.Cluster {
	return partrows.Cluster{
		Table:         thermostat.Table(),
		FeatureChoice: "at least one of HEAT and COOL is required unless AUTO is present and the device type fixes none",
	}
}

// Parts returns the required parts; the device type requires none.
func Parts() []composition.PartSpec {
	return []composition.PartSpec{}
}

// Servers returns the server clusters mounted on the root endpoint besides
// Descriptor.
func Servers() []composition.ServerRow {
	return partrows.Servers(identifyCluster().Table, thermostatCluster().Table)
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes and the rows of the Identify and Thermostat servers.
func Rows() []composition.AttributeRow {
	id, th := identifyCluster(), thermostatCluster()
	rows := partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision, []uint32{0x001D, id.Table.ClusterID, th.Table.ClusterID}, false, nil)
	rows = append(rows, id.Rows("")...)
	rows = append(rows, th.Rows("")...)
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
