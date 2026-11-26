// Package aggregator is the bridge profile of Matter 1.6.1. The bridge itself
// composes as the Aggregator device type; every composed resource is one
// Bridged Node part under it. The SDK assembler supplies the Aggregator
// endpoint, its Descriptor and the Bridged Device Basic Information of each
// part, so the profile mounts no cluster of its own. A service kind of the
// target with no composed profile and no fold is a named loss.
package aggregator

import (
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Device type identities.
const (
	DeviceTypeID           uint32 = 0x000E
	DeviceTypeRevision     uint32 = 2
	BridgedNodeDeviceType  uint32 = 0x0013
	BridgedNodeRevision    uint32 = 3
	DescriptorClusterID    uint32 = 0x001D
	BridgedBasicClusterID  uint32 = 0x0039
	AdminCommissioningID   uint32 = 0x003C
	PowerSourceClusterID   uint32 = 0x002F
	PowerSourceConfigID    uint32 = 0x002E
	EcosystemInformationID uint32 = 0x0750
)

// Subject is the profile key. The bridge is not a SemReg resource kind.
const Subject = "bridge"

// Requirements returns the requirements tree of the Aggregator and of the
// Bridged Node device type placed under it. Mounted marks what the node
// provides; SDKSupplied marks what the SDK assembler supplies.
func Requirements() []partrows.Requirement {
	return []partrows.Requirement{
		{Element: "server_cluster", ID: DescriptorClusterID, Name: "Descriptor", SDKSupplied: true, Mounted: true, Children: []partrows.Requirement{
			{Element: "attribute", ID: 0x0000, Name: "DeviceTypeList", SDKSupplied: true, Mounted: true},
		}},
		{Element: "server_cluster", ID: 0x0025, Name: "Actions", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0003, Name: "Identify", Conformance: "O"},
		{Element: "server_cluster", ID: 0x0751, Name: "Commissioner Control", Conformance: "FabricSynchronization", Note: "conditional on a node-level condition; not mounted"},
		{Element: "device_type", ID: BridgedNodeDeviceType, Name: "Bridged Node", Conformance: "M", Mounted: true, Note: "one part per composed resource; placement is the Aggregator PartsList", Children: []partrows.Requirement{
			{Element: "server_cluster", ID: DescriptorClusterID, Name: "Descriptor", SDKSupplied: true, Mounted: true},
			{Element: "server_cluster", ID: BridgedBasicClusterID, Name: "Bridged Device Basic Information", Conformance: "M", SDKSupplied: true, Mounted: true},
			{Element: "server_cluster", ID: PowerSourceConfigID, Name: "Power Source Configuration", Conformance: "BridgedPowerSourceInfo, D", Note: "conditional; not mounted"},
			{Element: "server_cluster", ID: PowerSourceClusterID, Name: "Power Source", Conformance: "BridgedPowerSourceInfo", Note: "conditional; not mounted"},
			{Element: "server_cluster", ID: AdminCommissioningID, Name: "Administrator Commissioning", Conformance: "FabricSynchronizedNode", Note: "conditional on a node-level condition; not mounted"},
			{Element: "server_cluster", ID: EcosystemInformationID, Name: "Ecosystem Information", Conformance: "FabricSynchronizedNode, O", Note: "not mounted"},
			{Element: "device_type", ID: 0x0011, Name: "Power Source", Conformance: "O"},
		}},
	}
}

// Rows returns the Descriptor value the profile fixes for the Aggregator
// endpoint: its device type and revision. The parts list is assigned by the
// node from the composed resources, so no value is declared for it.
func Rows() []composition.AttributeRow {
	return []composition.AttributeRow{{
		ClusterID: DescriptorClusterID, ClusterName: "Descriptor", AttributeID: 0x0000, Name: "DeviceTypeList",
		MatterType: "list", Mandatory: true, Origin: composition.OriginDeclared,
		Declared: []any{map[string]any{"device_type": "0x000E", "revision": int(DeviceTypeRevision)}},
		Detail:   "fixed by the device type",
	}}
}

// Parts returns the typed parts of the Aggregator. The Bridged Node parts are
// the endpoints of the composed resources, not rows of this profile.
func Parts() []composition.PartSpec { return []composition.PartSpec{} }

// Folds returns the declared losses of the bridge itself: none.
func Folds() []composition.Fold { return []composition.Fold{} }

// MissingRows returns the report names of mandatory rows with no source.
func MissingRows() []string { return partrows.Missing(Rows(), Parts()) }

// loss text is the target's own statement for a service row.
const lossText = "a service is not a kernel-v1 requested item; its subject has no allocated endpoint in this target and no endpoint correspondence is inferred"

// NoCounterpart returns the service kinds the target lists as unsupported that
// no profile composes and no fold absorbs. Each is a named identity loss; the
// values stay in SemReg, MCP and GraphQL.
func NoCounterpart() []composition.Fold {
	services := []string{
		"energy.service.subject", "energy.service.tariff",
		"equipment.service.resource", "equipment.service.unit",
		"evse.service.meter", "evse.service.session", "evse.service.vehicle",
		"infrastructure.service.feeder", "infrastructure.service.line_pair", "infrastructure.service.load",
		"infrastructure.service.site",
		"pv.service.array", "pv.service.string",
		"storage.service.cell", "storage.service.interface", "storage.service.module",
		"storage.service.pack", "storage.service.string", "storage.service.system",
		"thermal.service.air_filter", "thermal.service.circuit", "thermal.service.compressor", "thermal.service.detector",
		"thermal.service.fan", "thermal.service.function", "thermal.service.refrigerant_circuit", "thermal.service.room",
		"thermal.service.sensor", "thermal.service.solar_circuit", "thermal.service.solar_collector", "thermal.service.system",
		"thermal.service.terminal_unit", "thermal.service.thermal_store", "thermal.service.valve", "thermal.service.ventilation",
		"thermal.service.water_tank",
	}
	out := make([]composition.Fold, len(services))
	for i, id := range services {
		out[i] = composition.Fold{Name: id, Kind: "identity", SourceItems: []string{id}, Declared: true, Description: lossText}
	}
	return out
}
