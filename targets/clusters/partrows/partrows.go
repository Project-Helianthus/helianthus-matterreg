// Package partrows builds the typed attribute rows of one mounted cluster of a
// profile part from the cluster table and the requirements of the device type.
// Nothing is invented: a mandatory attribute with no transformed row and no
// value the data model fixes stays missing with the target's reason named.
package partrows

import (
	"fmt"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// UnmappedReason is the target's reason for an element with no transformed row.
const UnmappedReason = "matter.unmapped.v1"

// Requirement is one node of a device type's requirements tree.
type Requirement struct {
	Element     string
	ID          uint32
	Name        string
	Conformance string
	Min         int
	Mounted     bool
	SDKSupplied bool
	Note        string
	Children    []Requirement
}

// Cluster is one cluster mounted at a part. Features are the feature codes the
// device type requires; Attributes are the attribute names it requires beyond
// the unconditional ones. FeatureChoice, when set, is the reason the
// FeatureMap stays missing because a choice between features is left open.
type Cluster struct {
	Table         clusters.Table
	Features      []string
	Attributes    []string
	FeatureChoice string
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// Mandatory reports whether attribute a is mandatory for this mounting.
func (c Cluster) Mandatory(a clusters.Attribute) bool {
	return a.Mandatory || contains(c.Attributes, a.Name) || contains(c.Features, a.Conformance)
}

func (c Cluster) featureMap() uint32 {
	var bits uint32
	for _, f := range c.Table.Features {
		if contains(c.Features, f.Code) {
			bits |= 1 << f.Bit
		}
	}
	return bits
}

func hex(id uint32) string { return fmt.Sprintf("0x%04X", id) }

func row(part string, c clusters.Table, a clusters.Attribute) composition.AttributeRow {
	return composition.AttributeRow{
		PartKey: part, ClusterID: c.ClusterID, ClusterName: c.Name, AttributeID: a.ID, Name: a.Name,
		MatterType: a.MatterType, Nullable: a.Nullable, Mandatory: true,
		Origin: composition.OriginMissing, Reason: UnmappedReason,
		Detail: "the target has no transformed row; no default or derived value is created",
	}
}

var accuracyFields = []struct{ name, matterType string }{
	{"MeasurementType", "enum16"}, {"Measured", "bool"}, {"MinMeasuredValue", "int64"},
	{"MaxMeasuredValue", "int64"}, {"AccuracyRanges", "list"}, {"RangeMin", "int64"}, {"RangeMax", "int64"},
}

// Rows returns the mandatory attribute rows of the cluster at part.
func (c Cluster) Rows(part string) []composition.AttributeRow {
	var out []composition.AttributeRow
	for _, a := range c.Table.Attributes {
		if !c.Mandatory(a) {
			continue
		}
		r := row(part, c.Table, a)
		switch a.ID {
		case 0xFFFD:
			r.Origin, r.Reason, r.Detail, r.Declared = composition.OriginDeclared, "", "fixed by the data model", int(c.Table.Revision)
		case 0xFFFC:
			if c.FeatureChoice != "" {
				r.Detail = c.FeatureChoice
			} else {
				r.Origin, r.Reason, r.Detail, r.Declared = composition.OriginDeclared, "", "the features the device type requires and no other", int(c.featureMap())
			}
		}
		out = append(out, r)
		if a.Name == "Accuracy" {
			for _, f := range accuracyFields {
				fr := r
				fr.Field, fr.MatterType, fr.Nullable = f.name, f.matterType, false
				out = append(out, fr)
			}
		}
	}
	return out
}

// Descriptor returns the Descriptor rows of a part. Descriptor is supplied by
// the SDK; these rows are the values the profile fixes for it.
func Descriptor(part string, deviceType uint32, revision uint32, servers []uint32, taglist bool, parts []string) []composition.AttributeRow {
	base := composition.AttributeRow{PartKey: part, ClusterID: 0x001D, ClusterName: "Descriptor", Mandatory: true, Origin: composition.OriginDeclared}
	mk := func(id uint32, name, typ string, declared any, detail string) composition.AttributeRow {
		r := base
		r.AttributeID, r.Name, r.MatterType, r.Declared, r.Detail = id, name, typ, declared, detail
		return r
	}
	serverList := make([]any, len(servers))
	for i, s := range servers {
		serverList[i] = hex(s)
	}
	partList := make([]any, len(parts))
	for i, p := range parts {
		partList[i] = p
	}
	var feature int
	if taglist {
		feature = 1
	}
	out := []composition.AttributeRow{
		mk(0x0000, "DeviceTypeList", "list", []any{map[string]any{"device_type": hex(deviceType), "revision": int(revision)}}, "fixed by the device type"),
		mk(0x0001, "ServerList", "list", serverList, "fixed by the mounted set"),
		mk(0x0002, "ClientList", "list", []any{}, "no client cluster is mounted"),
		mk(0x0003, "PartsList", "list", partList, "part roles of the profile; endpoint numbers are assigned by the node"),
		mk(0xFFFC, "FeatureMap", "bitmap32", feature, "TAGLIST when the device type requires it"),
		mk(0xFFFD, "ClusterRevision", "uint16", 3, "fixed by the data model"),
	}
	if taglist {
		r := mk(0x0004, "TagList", "list", nil, "the device type requires TAGLIST and the target maps no tag; no value is created")
		r.Origin, r.Reason = composition.OriginMissing, UnmappedReason
		out = append(out, r)
	}
	return out
}

// Missing returns the report names of the missing mandatory rows of a part
// list, including the root rows.
func Missing(root []composition.AttributeRow, parts []composition.PartSpec) []string {
	var names []string
	var walk func(rows []composition.AttributeRow)
	walk = func(rows []composition.AttributeRow) {
		for _, r := range rows {
			if r.Origin == composition.OriginMissing {
				names = append(names, r.ReportName())
			}
		}
	}
	walk(root)
	var visit func([]composition.PartSpec)
	visit = func(ps []composition.PartSpec) {
		for _, p := range ps {
			walk(p.Attributes)
			visit(p.Children)
		}
	}
	visit(parts)
	return names
}

// Servers returns the server rows of the mounted clusters.
func Servers(cs ...clusters.Table) []composition.ServerRow {
	out := make([]composition.ServerRow, len(cs))
	for i, t := range cs {
		out[i] = composition.ServerRow{ClusterID: t.ClusterID, Name: t.Name, Revision: uint32(t.Revision)}
	}
	return out
}
