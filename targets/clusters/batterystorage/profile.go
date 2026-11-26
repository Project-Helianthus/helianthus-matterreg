// Package batterystorage is the Battery Storage profile of Matter 1.6.1 for
// the battery resource kind. The device type requires two Electrical Sensor
// parts (instance 1 with ALTC, instance 2 with DIRC) and two Power Source
// parts (instance 1 with WIRED, instance 2 with BAT), all scoped under the one
// resource root. The target maps no transformed row for this kind, so every
// dynamic mandatory element is missing with the reason named and nothing is
// invented.
package batterystorage

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
	DeviceTypeID                 uint32 = 0x0018
	DeviceTypeRevision           uint32 = 2
	PowerSourceDeviceTypeID      uint32 = 0x0011
	PowerSourceRevision          uint32 = 1
	ElectricalSensorDeviceTypeID uint32 = 0x0510
	ElectricalSensorRevision     uint32 = 1
)

// Part roles, one per required device-type instance.
const (
	RoleElectricalSensorAC = "electrical_sensor_1"
	RoleElectricalSensorDC = "electrical_sensor_2"
	RolePowerSourceWired   = "power_source_1"
	RolePowerSourceBattery = "power_source_2"
)

var batteryAttributes = []string{
	"BatVoltage", "BatPercentRemaining", "BatTimeRemaining", "ActiveBatFaults",
	"BatCapacity", "BatTimeToFullCharge", "BatChargingCurrent", "ActiveBatChargeFaults",
}

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.battery"}
}

func tagList() partrows.Requirement {
	return partrows.Requirement{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true, Children: []partrows.Requirement{
		{Element: "feature", Name: "TAGLIST", Conformance: "M", Mounted: true},
	}}
}

func sensor(feature string) partrows.Requirement {
	return partrows.Requirement{Element: "device_type", ID: ElectricalSensorDeviceTypeID, Name: "Electrical Sensor", Conformance: "M", Min: 2, Mounted: true, Children: []partrows.Requirement{
		{Element: "server_cluster", ID: 0x0090, Name: "Electrical Power Measurement", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
			{Element: "feature", Name: feature, Conformance: "M", Mounted: true},
			{Element: "attribute", ID: 0x0004, Name: "Voltage", Conformance: "M", Mounted: true},
			{Element: "attribute", ID: 0x0005, Name: "ActiveCurrent", Conformance: "M", Mounted: true},
		}},
		{Element: "server_cluster", ID: 0x0091, Name: "Electrical Energy Measurement", Conformance: "M", Mounted: true, Children: []partrows.Requirement{
			{Element: "feature", Name: "EXPE", Conformance: "M", Mounted: true},
		}},
		tagList(),
		{Element: "server_cluster", ID: 0x009C, Name: "Power Topology", Conformance: "M", Mounted: true, Note: "required by the Electrical Sensor device type itself"},
	}}
}

func source(feature string, attributes ...string) partrows.Requirement {
	children := []partrows.Requirement{{Element: "feature", Name: feature, Conformance: "M", Mounted: true}}
	for _, name := range attributes {
		children = append(children, partrows.Requirement{Element: "attribute", Name: name, Conformance: "M", Mounted: true})
	}
	return partrows.Requirement{Element: "device_type", ID: PowerSourceDeviceTypeID, Name: "Power Source", Conformance: "M", Min: 2, Mounted: true, Children: []partrows.Requirement{
		tagList(),
		{Element: "server_cluster", ID: 0x002F, Name: "Power Source", Mounted: true, Children: children},
	}}
}

// Requirements returns the requirements tree of the device type, instances in
// the order the data model lists them.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: 0x001D, Name: "Descriptor", SDKSupplied: true, Mounted: true},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		sensor("ALTC"),
		sensor("DIRC"),
		source("WIRED"),
		source("BAT", batteryAttributes...),
		{Element: "device_type", ID: 0x0017, Name: "Solar Power", Conformance: "O"},
		{Element: "device_type", ID: 0x0302, Name: "Temperature Sensor", Conformance: "O"},
	}
}

func sensorPart(role, feature string) composition.PartSpec {
	ep := partrows.Cluster{Table: electricalpowermeasurement.Table(), Features: []string{feature}, Attributes: []string{"Voltage", "ActiveCurrent"}}
	ee := partrows.Cluster{
		Table: electricalenergymeasurement.Table(), Features: []string{"EXPE"},
		FeatureChoice: "EXPE is required; at least one of CUME and PERE is required and the device type fixes neither",
	}
	pt := partrows.Cluster{
		Table:         powertopology.Table(),
		FeatureChoice: "exactly one of NODE, TREE and SET is required and nothing fixes which",
	}
	rows := partrows.Descriptor(role, ElectricalSensorDeviceTypeID, ElectricalSensorRevision,
		[]uint32{0x001D, ep.Table.ClusterID, ee.Table.ClusterID, pt.Table.ClusterID}, true, nil)
	rows = append(rows, ep.Rows(role)...)
	rows = append(rows, ee.Rows(role)...)
	rows = append(rows, pt.Rows(role)...)
	return composition.PartSpec{
		Role: role, DeviceTypeID: ElectricalSensorDeviceTypeID, Revision: ElectricalSensorRevision, Count: 1,
		Attributes: rows, Servers: partrows.Servers(ep.Table, ee.Table, pt.Table),
	}
}

func sourcePart(role string, c partrows.Cluster) composition.PartSpec {
	rows := partrows.Descriptor(role, PowerSourceDeviceTypeID, PowerSourceRevision, []uint32{0x001D, c.Table.ClusterID}, true, nil)
	rows = append(rows, c.Rows(role)...)
	return composition.PartSpec{
		Role: role, DeviceTypeID: PowerSourceDeviceTypeID, Revision: PowerSourceRevision, Count: 1,
		Attributes: rows, Servers: partrows.Servers(c.Table),
	}
}

// Parts returns the four required parts in their declared geometry. The
// battery Power Source mounts RECHG as well as BAT: the device type requires
// BatTimeToFullCharge, BatChargingCurrent and ActiveBatChargeFaults, which the
// data model conforms to RECHG, though it lists only BAT as a feature.
func Parts() []composition.PartSpec {
	return []composition.PartSpec{
		sensorPart(RoleElectricalSensorAC, "ALTC"),
		sensorPart(RoleElectricalSensorDC, "DIRC"),
		sourcePart(RolePowerSourceWired, partrows.Cluster{Table: powersource.Table(), Features: []string{"WIRED"}}),
		sourcePart(RolePowerSourceBattery, partrows.Cluster{
			Table: powersource.Table(), Features: []string{"BAT", "RECHG"}, Attributes: batteryAttributes,
		}),
	}
}

// Rows returns the mandatory rows of the root endpoint: the Descriptor values
// the profile fixes.
func Rows() []composition.AttributeRow {
	return partrows.Descriptor("", DeviceTypeID, DeviceTypeRevision, []uint32{0x001D}, false,
		[]string{RoleElectricalSensorAC, RoleElectricalSensorDC, RolePowerSourceWired, RolePowerSourceBattery})
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	return partrows.Missing(Rows(), Parts())
}

// Folds returns the declared losses. The target names no fold for this kind.
func Folds() []composition.Fold {
	return []composition.Fold{}
}
