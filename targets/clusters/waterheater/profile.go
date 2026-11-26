// Package waterheater is the Water Heater profile of Matter 1.6.1 for the
// domestic hot water resource kind. The device type requires the Water Heater
// Management, Water Heater Mode and Thermostat servers on the root, with the
// Thermostat feature HEAT. The target maps no transformed row for this kind,
// so every dynamic mandatory element is missing with the reason named and
// nothing is invented.
package waterheater

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/thermostat"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/waterheatermanagement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/waterheatermode"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identity.
const (
	DeviceTypeID       uint32 = 0x050F
	DeviceTypeRevision uint32 = 1
)

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.domestic_hot_water"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
// Every nested device type is optional or described elsewhere in the data
// model, so none is mounted.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0094, Name: "Water Heater Management", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x009E, Name: "Water Heater Mode", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0201, Name: "Thermostat", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
			{Element: "feature", Name: "HEAT", Conformance: "M", Mounted: true},
		}},
		{Element: "device_type", ID: 0x0011, Name: "Power Source", Conformance: "O"},
		{Element: "device_type", ID: 0x0302, Name: "Temperature Sensor", Conformance: "O"},
		{Element: "device_type", ID: 0x050D, Name: "Device Energy Management", Conformance: "O", Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0098, Name: "Device Energy Management", Children: []partrows.Requirement{
				{Element: "feature", Name: "PFR", Conformance: "M"},
			}},
		}},
		{Element: "device_type", ID: 0x0510, Name: "Electrical Sensor", Conformance: "desc", Note: "conformance is described elsewhere in the data model; not mounted"},
	}
}

func management() partrows.Cluster {
	return partrows.Cluster{Table: waterheatermanagement.Table()}
}

func mode() partrows.Cluster {
	return partrows.Cluster{Table: waterheatermode.Table()}
}

func thermostatCluster() partrows.Cluster {
	return partrows.Cluster{Table: thermostat.Table(), Features: []string{"HEAT"}}
}

// Parts returns the required parts; no nested device type is required.
func Parts() []composition.PartSpec {
	return []composition.PartSpec{}
}

// Servers returns the server clusters mounted on the root endpoint besides
// Descriptor.
func Servers() []composition.ServerRow {
	return partrows.Servers(management().Table, mode().Table, thermostatCluster().Table)
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes and the rows of the three servers.
func Rows() []composition.AttributeRow {
	mg, md, th := management(), mode(), thermostatCluster()
	rows := partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision,
		[]uint32{0x001D, mg.Table.ClusterID, md.Table.ClusterID, th.Table.ClusterID}, false, nil)
	rows = append(rows, mg.Rows("")...)
	rows = append(rows, md.Rows("")...)
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
