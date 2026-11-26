// Package heatpump is the Heat Pump profile of Matter 1.6.1 for the heat
// generator resource kind. The device type requires a Power Source part, a
// Device Energy Management part and at least one Electrical Sensor part, all
// scoped under the one resource root; the root itself serves no cluster
// besides Descriptor. The target maps no transformed row for this kind, so
// every dynamic mandatory element is missing with the reason named and nothing
// is invented. The model gives no boiler device type; a heat generator that is
// a boiler is not distinguished by any source here.
package heatpump

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/deviceenergymanagement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalenergymeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powersource"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powertopology"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identities.
const (
	DeviceTypeID                 uint32 = 0x0309
	DeviceTypeRevision           uint32 = 1
	PowerSourceDeviceTypeID      uint32 = 0x0011
	PowerSourceRevision          uint32 = 1
	DeviceEnergyMgmtDeviceTypeID uint32 = 0x050D
	DeviceEnergyMgmtRevision     uint32 = 3
	ElectricalSensorDeviceTypeID uint32 = 0x0510
	ElectricalSensorRevision     uint32 = 1
)

// Part roles, one per required device-type instance.
const (
	RolePowerSource      = "power_source"
	RoleDeviceEnergyMgmt = "device_energy_management"
	RoleElectricalSensor = "electrical_sensor"
)

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.heat_generator"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		{Element: "client_cluster", ID: 0x0201, Name: "Thermostat", Conformance: "O"},
		{Element: "device_type", ID: PowerSourceDeviceTypeID, Name: "Power Source", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x002F, Name: "Power Source", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "WIRED", Conformance: "M", Mounted: true},
			}},
			{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "TAGLIST", Conformance: "M", Mounted: true},
			}},
		}},
		{Element: "device_type", ID: 0x0301, Name: "Thermostat", Conformance: "O", Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0041, Name: "User Label", Conformance: "M"},
			{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", Children: []partrows.Requirement{
				{Element: "feature", Name: "TAGLIST", Conformance: "M"},
			}},
		}},
		{Element: "device_type", ID: 0x0302, Name: "Temperature Sensor", Conformance: "O", Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", Children: []partrows.Requirement{
				{Element: "feature", Name: "TAGLIST", Conformance: "M"},
			}},
		}},
		{Element: "device_type", ID: DeviceEnergyMgmtDeviceTypeID, Name: "Device Energy Management", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0098, Name: "Device Energy Management", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "PA", Conformance: "M", Mounted: true},
			}},
		}},
		{Element: "device_type", ID: 0x050F, Name: "Water Heater", Conformance: "O"},
		{Element: "device_type", ID: ElectricalSensorDeviceTypeID, Name: "Electrical Sensor", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0090, Name: "Electrical Power Measurement", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "ALTC", Conformance: "M", Mounted: true},
				{Element: "attribute", ID: 0x0004, Name: "Voltage", Conformance: "M", Mounted: true},
				{Element: "attribute", ID: 0x0005, Name: "ActiveCurrent", Conformance: "M", Mounted: true},
			}},
			{Element: "server_cluster", ID: 0x0091, Name: "Electrical Energy Measurement", Conformance: "M", Mounted: true},
			{Element: "server_cluster", ID: 0x009C, Name: "Power Topology", Conformance: "M", Mounted: true, Note: "required by the Electrical Sensor device type itself"},
		}},
	}
}

func powerSource() partrows.Cluster {
	return partrows.Cluster{Table: powersource.Table(), Features: []string{"WIRED"}}
}

func deviceEnergy() partrows.Cluster {
	return partrows.Cluster{Table: deviceenergymanagement.Table(), Features: []string{"PA"}, Attributes: []string{"OptOutState"}}
}

func electricalPower() partrows.Cluster {
	return partrows.Cluster{Table: electricalpowermeasurement.Table(), Features: []string{"ALTC"}, Attributes: []string{"Voltage", "ActiveCurrent"}}
}

func electricalEnergy() partrows.Cluster {
	return partrows.Cluster{
		Table:         electricalenergymeasurement.Table(),
		FeatureChoice: "at least one of CUME and PERE is required and the device type fixes neither",
	}
}

func powerTopology() partrows.Cluster {
	return partrows.Cluster{
		Table:         powertopology.Table(),
		FeatureChoice: "exactly one of NODE, TREE and SET is required and nothing fixes which",
	}
}

// Parts returns the required parts in their declared geometry, each with its
// nested clusters, rows and servers.
func Parts() []composition.PartSpec {
	ps, dem := powerSource(), deviceEnergy()
	ep, ee, pt := electricalPower(), electricalEnergy(), powerTopology()
	psRows := partrows.Descriptor(RolePowerSource, PowerSourceDeviceTypeID, PowerSourceRevision, []uint32{0x001D, ps.Table.ClusterID}, true, nil)
	psRows = append(psRows, ps.Rows(RolePowerSource)...)
	demRows := partrows.Descriptor(RoleDeviceEnergyMgmt, DeviceEnergyMgmtDeviceTypeID, DeviceEnergyMgmtRevision, []uint32{0x001D, dem.Table.ClusterID}, false, nil)
	demRows = append(demRows, dem.Rows(RoleDeviceEnergyMgmt)...)
	esRows := partrows.Descriptor(RoleElectricalSensor, ElectricalSensorDeviceTypeID, ElectricalSensorRevision,
		[]uint32{0x001D, ep.Table.ClusterID, ee.Table.ClusterID, pt.Table.ClusterID}, false, nil)
	esRows = append(esRows, ep.Rows(RoleElectricalSensor)...)
	esRows = append(esRows, ee.Rows(RoleElectricalSensor)...)
	esRows = append(esRows, pt.Rows(RoleElectricalSensor)...)
	return []composition.PartSpec{
		{Role: RolePowerSource, DeviceTypeID: PowerSourceDeviceTypeID, Revision: PowerSourceRevision, Count: 1,
			Attributes: psRows, Servers: partrows.Servers(ps.Table)},
		{Role: RoleDeviceEnergyMgmt, DeviceTypeID: DeviceEnergyMgmtDeviceTypeID, Revision: DeviceEnergyMgmtRevision, Count: 1,
			Attributes: demRows, Servers: partrows.Servers(dem.Table)},
		{Role: RoleElectricalSensor, DeviceTypeID: ElectricalSensorDeviceTypeID, Revision: ElectricalSensorRevision, Count: 1,
			Attributes: esRows, Servers: partrows.Servers(ep.Table, ee.Table, pt.Table)},
	}
}

// Servers returns the server clusters mounted on the root endpoint besides
// Descriptor; the device type requires none.
func Servers() []composition.ServerRow {
	return partrows.Servers()
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes.
func Rows() []composition.AttributeRow {
	return partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision, []uint32{0x001D},
		false, []string{RolePowerSource, RoleDeviceEnergyMgmt, RoleElectricalSensor})
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	return partrows.Missing(Rows(), Parts())
}

// Folds returns the declared losses. The target names no fold for this kind.
func Folds() []composition.Fold {
	return []composition.Fold{}
}
