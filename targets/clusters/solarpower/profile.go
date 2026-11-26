// Package solarpower is the Solar Power profile of Matter 1.6.1 for the
// inverter and PV system resource kinds. The device type requires a Power
// Source part and an Electrical Sensor part, both scoped under the one
// resource root. The target maps no transformed row for these kinds, so every
// dynamic mandatory element is missing with the reason named and nothing is
// invented.
package solarpower

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalenergymeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powersource"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powertopology"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identities.
const (
	DeviceTypeID                 uint32 = 0x0017
	DeviceTypeRevision           uint32 = 1
	PowerSourceDeviceTypeID      uint32 = 0x0011
	PowerSourceDeviceTypeRev     uint32 = 1
	ElectricalSensorDeviceTypeID uint32 = 0x0510
	ElectricalSensorRevision     uint32 = 1
)

// Part roles.
const (
	RolePowerSource      = "power_source"
	RoleElectricalSensor = "electrical_sensor"
)

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.inverter", "resource.pv_system"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		{Element: "device_type", ID: PowerSourceDeviceTypeID, Name: "Power Source", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x002F, Name: "Power Source", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "WIRED", Conformance: "M", Mounted: true},
			}},
			{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "TAGLIST", Conformance: "M", Mounted: true},
			}},
		}},
		{Element: "device_type", ID: 0x0302, Name: "Temperature Sensor", Conformance: "O"},
		{Element: "device_type", ID: 0x050D, Name: "Device Energy Management", Conformance: "desc", Note: "conformance is described elsewhere in the data model; not mounted"},
		{Element: "device_type", ID: ElectricalSensorDeviceTypeID, Name: "Electrical Sensor", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0041, Name: "User Label", Conformance: "desc"},
			{Element: "server_cluster", ID: 0x0090, Name: "Electrical Power Measurement", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
				{Element: "attribute", ID: 0x0004, Name: "Voltage", Conformance: "M", Mounted: true},
				{Element: "attribute", ID: 0x0005, Name: "ActiveCurrent", Conformance: "M", Mounted: true},
			}},
			{Element: "server_cluster", ID: 0x0091, Name: "Electrical Energy Measurement", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "EXPE", Conformance: "M", Mounted: true},
			}},
			{Element: "server_cluster", ID: 0x009C, Name: "Power Topology", Conformance: "M", Mounted: true, Note: "required by the Electrical Sensor device type itself"},
		}},
	}
}

func powerSource() partrows.Cluster {
	return partrows.Cluster{Table: powersource.Table(), Features: []string{"WIRED"}}
}

func electricalPower() partrows.Cluster {
	return partrows.Cluster{
		Table: electricalpowermeasurement.Table(), Attributes: []string{"Voltage", "ActiveCurrent"},
		FeatureChoice: "at least one of DIRC and ALTC is required and the device type fixes neither",
	}
}

func electricalEnergy() partrows.Cluster {
	return partrows.Cluster{
		Table: electricalenergymeasurement.Table(), Features: []string{"EXPE"},
		FeatureChoice: "EXPE is required; at least one of CUME and PERE is required and the device type fixes neither",
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
	ps, ep, ee, pt := powerSource(), electricalPower(), electricalEnergy(), powerTopology()
	psRows := partrows.Descriptor(RolePowerSource, PowerSourceDeviceTypeID, PowerSourceDeviceTypeRev, []uint32{0x001D, ps.Table.ClusterID}, true, nil)
	psRows = append(psRows, ps.Rows(RolePowerSource)...)
	esRows := partrows.Descriptor(RoleElectricalSensor, ElectricalSensorDeviceTypeID, ElectricalSensorRevision,
		[]uint32{0x001D, ep.Table.ClusterID, ee.Table.ClusterID, pt.Table.ClusterID}, false, nil)
	esRows = append(esRows, ep.Rows(RoleElectricalSensor)...)
	esRows = append(esRows, ee.Rows(RoleElectricalSensor)...)
	esRows = append(esRows, pt.Rows(RoleElectricalSensor)...)
	return []composition.PartSpec{
		{Role: RolePowerSource, DeviceTypeID: PowerSourceDeviceTypeID, Revision: PowerSourceDeviceTypeRev, Count: 1,
			Attributes: psRows, Servers: partrows.Servers(ps.Table)},
		{Role: RoleElectricalSensor, DeviceTypeID: ElectricalSensorDeviceTypeID, Revision: ElectricalSensorRevision, Count: 1,
			Attributes: esRows, Servers: partrows.Servers(ep.Table, ee.Table, pt.Table)},
	}
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes. The device type has no other server cluster at the root.
func Rows() []composition.AttributeRow {
	return partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision, []uint32{0x001D},
		false, []string{RolePowerSource, RoleElectricalSensor})
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	return partrows.Missing(Rows(), Parts())
}

// Folds returns the declared losses: DC inputs and phases become measurement
// entries of the parent endpoint and their values stay in SemReg, MCP and
// GraphQL.
func Folds() []composition.Fold {
	return []composition.Fold{
		{
			Name: "dc_input", Kind: "identity", SourceItems: []string{"pv.dimension.input"}, Declared: true,
			Description: "DC inputs fold into measurement entries of the parent endpoint; the input qualifier is not retained as an endpoint identity",
		},
		{
			Name: "phase", Kind: "identity", SourceItems: []string{"pv.dimension.phase"}, Declared: true,
			Description: "phases fold into measurement entries of the parent endpoint; the phase qualifier is not retained as an endpoint identity",
		},
	}
}
