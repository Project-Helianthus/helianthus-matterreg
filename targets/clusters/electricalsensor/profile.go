// Package electricalsensor is the Electrical Sensor profile of Matter 1.6.1
// for the meter, grid connection and electrical circuit resource kinds.
// The target maps no transformed row for these kinds, so every dynamic
// mandatory element is missing with the reason named and nothing is invented.
package electricalsensor

import "github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"

// DeviceTypeID and DeviceTypeRevision identify the Electrical Sensor device type.
const (
	DeviceTypeID       uint32 = 0x0510
	DeviceTypeRevision uint16 = 1
)

// UnmappedReason is the target's reason for an element with no transformed row.
const UnmappedReason = "matter.unmapped.v1"

// Kinds returns the resource kinds this profile composes.
func Kinds() []string {
	return []string{"resource.meter", "resource.grid_connection", "resource.electrical_circuit"}
}

// Requirement is one server cluster of the device type's requirements tree.
type Requirement struct {
	ClusterID    uint32
	Name         string
	Conformance  string
	Mounted      bool
	SDKSupplied  bool
	ChoiceReason string
}

// Requirements returns the server clusters the device type requires and the
// mounted set. The device type has no nested device-type requirement.
func Requirements() []Requirement {
	return []Requirement{
		{ClusterID: 0x001D, Name: "Descriptor", Conformance: "M", Mounted: true, SDKSupplied: true},
		{ClusterID: 0x0090, Name: "Electrical Power Measurement", Conformance: "O.a+", Mounted: true, ChoiceReason: "satisfies the at-least-one choice of Electrical Power Measurement and Electrical Energy Measurement"},
		{ClusterID: 0x0091, Name: "Electrical Energy Measurement", Conformance: "O.a+"},
		{ClusterID: 0x009C, Name: "Power Topology", Conformance: "M", Mounted: true},
	}
}

// Rows returns the mandatory attribute rows of the mounted clusters. Dynamic
// rows carry the target's unmapped reason.
func Rows() []pilot.AttributeRow {
	var out []pilot.AttributeRow
	for _, row := range pilot.Rows() {
		if !row.Mandatory {
			continue
		}
		if row.Origin == pilot.OriginMissing && row.Reason == "" && row.AttributeID < 0xFFF0 {
			row.Reason = UnmappedReason
		}
		out = append(out, row)
	}
	return out
}

// MissingRows returns the report names of the mandatory rows with no source.
func MissingRows() []string {
	var names []string
	for _, row := range Rows() {
		if row.Origin == pilot.OriginMissing {
			names = append(names, row.ReportName())
		}
	}
	return names
}

// Folds returns the declared losses: phases become measurement entries of the
// parent endpoint and their values stay in SemReg, MCP and GraphQL.
func Folds() []pilot.Fold {
	return []pilot.Fold{{
		Name:        "phase",
		Kind:        "identity",
		SourceItems: []string{"infrastructure.dimension.phase"},
		Description: "phases fold into measurement entries of the parent endpoint; the phase qualifier is not retained as an endpoint identity",
		Declared:    true,
	}}
}
