// Package evse is the EVSE profile of Matter 1.6.1 for the charging station
// and EVSE resource kinds. The device type requires a Power Source part, a
// Device Energy Management part and an Electrical Sensor part, all scoped
// under the one resource root, with the Energy EVSE and Energy EVSE Mode
// servers on the root itself. The target maps one transformed row for these
// kinds (evse.ac.current to Electrical Power Measurement ActiveCurrent), so
// every other dynamic mandatory element is missing with the reason named and
// nothing is invented. Connectors fold into the EVSE endpoint as a declared
// loss.
package evse

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/deviceenergymanagement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalenergymeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/energyevse"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/energyevsemode"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powersource"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powertopology"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

// Device type identities.
const (
	DeviceTypeID                 uint32 = 0x050C
	DeviceTypeRevision           uint32 = 2
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
	return []string{"resource.charging_station", "resource.evse"}
}

// Requirements returns the requirements tree of the device type. Mounted marks
// what the profile mounts; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0099, Name: "Energy EVSE", Conformance: "M", Mounted: true, Note: "the cluster itself requires the PREF feature"},
		{Element: "server_cluster", ID: 0x009D, Name: "Energy EVSE Mode", Conformance: "M", Mounted: true},
		{Element: "server_cluster", ID: 0x0402, Name: "Temperature Measurement", Conformance: "O"},
		{Element: "device_type", ID: PowerSourceDeviceTypeID, Name: "Power Source", Conformance: "M", Min: 1, Mounted: true},
		{Element: "device_type", ID: DeviceEnergyMgmtDeviceTypeID, Name: "Device Energy Management", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0098, Name: "Device Energy Management", Mounted: true, Children: []partrows.Requirement{
				{Element: "feature", Name: "PFR", Conformance: "M", Mounted: true},
				{Element: "feature", Name: "PA", Conformance: "desc", Note: "conformance is described elsewhere in the data model; not mounted"},
			}},
		}},
		{Element: "device_type", ID: ElectricalSensorDeviceTypeID, Name: "Electrical Sensor", Conformance: "M", Min: 1, Mounted: true, Children: []partrows.Requirement{
			{Element: "server_cluster", ID: 0x0090, Name: "Electrical Power Measurement", Conformance: "M", Mounted: true},
			{Element: "server_cluster", ID: 0x0091, Name: "Electrical Energy Measurement", Conformance: "M", Mounted: true},
			{Element: "server_cluster", ID: 0x009C, Name: "Power Topology", Conformance: "M", Mounted: true, Note: "required by the Electrical Sensor device type itself"},
		}},
	}
}

func energyEvse() partrows.Cluster {
	return partrows.Cluster{Table: energyevse.Table(), Features: []string{"PREF"}}
}

func energyEvseMode() partrows.Cluster {
	return partrows.Cluster{Table: energyevsemode.Table()}
}

func powerSource() partrows.Cluster {
	return partrows.Cluster{
		Table:         powersource.Table(),
		FeatureChoice: "exactly one of WIRED and BAT is required and the device type fixes neither",
	}
}

func deviceEnergy() partrows.Cluster {
	return partrows.Cluster{
		Table: deviceenergymanagement.Table(), Features: []string{"PFR"}, Attributes: []string{"Forecast"},
		FeatureChoice: "PFR is required; PA is described elsewhere in the data model and nothing here fixes it",
	}
}

func electricalPower() partrows.Cluster {
	return partrows.Cluster{
		Table:         electricalpowermeasurement.Table(),
		FeatureChoice: "at least one of DIRC and ALTC is required and the device type fixes neither",
	}
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

// activeCurrent is the one transformed row of the target for these kinds. It
// is copied from the pilot row, which carries the exact mapping, and placed on
// the Electrical Sensor part. The attribute is optional, so it never
// discharges a mandatory element.
func activeCurrent() composition.AttributeRow {
	for _, row := range pilot.Rows() {
		if row.ClusterID == 0x0090 && row.AttributeID == 0x0005 {
			return composition.AttributeRow{
				PartKey: RoleElectricalSensor, ClusterID: row.ClusterID, ClusterName: row.ClusterName, AttributeID: row.AttributeID,
				Name: row.Name, MatterType: row.MatterType, Nullable: row.Nullable, Mandatory: row.Mandatory,
				Origin: string(row.Origin), PackID: row.PackID, PackVersion: row.PackVersion, Element: row.Element,
				Dimension: row.Dimension, Encoder: row.Encoder, Reason: row.Reason,
			}
		}
	}
	return composition.AttributeRow{}
}

// Parts returns the required parts in their declared geometry, each with its
// nested clusters, rows and servers.
func Parts() []composition.PartSpec {
	ps, dem := powerSource(), deviceEnergy()
	ep, ee, pt := electricalPower(), electricalEnergy(), powerTopology()
	psRows := partrows.Descriptor(RolePowerSource, PowerSourceDeviceTypeID, PowerSourceRevision, []uint32{0x001D, ps.Table.ClusterID}, false, nil)
	psRows = append(psRows, ps.Rows(RolePowerSource)...)
	demRows := partrows.Descriptor(RoleDeviceEnergyMgmt, DeviceEnergyMgmtDeviceTypeID, DeviceEnergyMgmtRevision, []uint32{0x001D, dem.Table.ClusterID}, false, nil)
	demRows = append(demRows, dem.Rows(RoleDeviceEnergyMgmt)...)
	esRows := partrows.Descriptor(RoleElectricalSensor, ElectricalSensorDeviceTypeID, ElectricalSensorRevision,
		[]uint32{0x001D, ep.Table.ClusterID, ee.Table.ClusterID, pt.Table.ClusterID}, false, nil)
	esRows = append(esRows, ep.Rows(RoleElectricalSensor)...)
	esRows = append(esRows, activeCurrent())
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
// Descriptor.
func Servers() []composition.ServerRow {
	return partrows.Servers(energyEvse().Table, energyEvseMode().Table)
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes and the rows of the Energy EVSE and Energy EVSE Mode
// servers.
func Rows() []composition.AttributeRow {
	ev, mode := energyEvse(), energyEvseMode()
	rows := partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision, []uint32{0x001D, ev.Table.ClusterID, mode.Table.ClusterID},
		false, []string{RolePowerSource, RoleDeviceEnergyMgmt, RoleElectricalSensor})
	rows = append(rows, ev.Rows("")...)
	rows = append(rows, mode.Rows("")...)
	return rows
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	return partrows.Missing(Rows(), Parts())
}

// Folds returns the declared losses. Connectors fold into the EVSE endpoint
// and phases fold into the measurement entries of the Electrical Sensor part;
// their values stay in SemReg, MCP and GraphQL.
func Folds() []composition.Fold {
	return []composition.Fold{
		{
			Name: "connector", Kind: "identity", SourceItems: []string{"evse.dimension.connector"}, Declared: true,
			Description: "connectors fold into the EVSE endpoint; the connector qualifier is not retained as an endpoint identity",
		},
		{
			Name: "phase", Kind: "identity", SourceItems: []string{"evse.dimension.phase"}, Declared: true,
			Description: "phases fold into measurement entries of the Electrical Sensor part; the phase qualifier is not retained as an endpoint identity",
		},
	}
}
