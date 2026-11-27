// Package pilot is the reviewed Electrical Sensor composition for Matter 1.6.1.
// Attribute sources are the table below. The JSON document carries only the
// admission shape; it does not carry a second copy of this table.
package pilot

import _ "embed"

//go:embed profile.json
var document []byte

// ResourceKind is the catalog kind of the one transformed element's dimension.
const ResourceKind = "resource.phase"

// Origin is how a profile row obtains its value.
type Origin string

const (
	OriginSemreg   Origin = "semreg"
	OriginDeclared Origin = "declared"
	OriginMissing  Origin = "missing"
)

// AttributeRow is one Matter attribute, or one mandatory field of an attribute,
// and the source that feeds it.
type AttributeRow struct {
	ClusterID   uint32
	ClusterName string
	AttributeID uint32
	Name        string
	MatterType  string
	Nullable    bool
	Mandatory   bool
	Field       string
	Origin      Origin
	PackID      string
	PackVersion string
	Element     string
	Dimension   string
	Encoder     string
	Reason      string
	Declared    any
	Detail      string
}

// Fold is one component of the pilot resource folded into the endpoint.
type Fold struct {
	Name        string
	Kind        string
	SourceItems []string
	Description string
	Declared    bool
}

// Document returns a copy of the embedded admission profile.
func Document() []byte {
	out := make([]byte, len(document))
	copy(out, document)
	return out
}

// Rows returns a copy of the attribute source table.
func Rows() []AttributeRow {
	out := make([]AttributeRow, len(attributeRows))
	for i := range attributeRows {
		out[i] = attributeRows[i]
		out[i].Declared = copyDeclared(attributeRows[i].Declared)
	}
	return out
}

// Folds returns a copy of the declared component folds.
func Folds() []Fold {
	out := make([]Fold, len(folds))
	for i := range folds {
		out[i] = folds[i]
		out[i].SourceItems = append([]string(nil), folds[i].SourceItems...)
	}
	return out
}

// ReportName is the missing-row name used when a mandatory source is absent.
func (r AttributeRow) ReportName() string {
	name := r.ClusterName + "." + r.Name
	if r.Field != "" {
		name += "." + r.Field
	}
	return name
}

func copyDeclared(v any) any {
	switch n := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(n))
		for key, item := range n {
			out[key] = copyDeclared(item)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i := range n {
			out[i] = copyDeclared(n[i])
		}
		return out
	default:
		return v
	}
}

var folds = []Fold{{
	Name:        "phase",
	Kind:        "identity",
	SourceItems: []string{"evse.ac.current"},
	Description: "the phase qualifier is not retained as an endpoint identity",
	Declared:    true,
}}

// attributeRows lists every mandatory attribute of the mounted clusters and the
// one transformed optional attribute. A value is declared only where the
// pinned device model fixes it. A mandatory attribute with no transformed
// element and no fixed value stays missing.
var attributeRows = []AttributeRow{
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0000, Name: "DeviceTypeList", MatterType: "list", Mandatory: true, Origin: OriginDeclared, Declared: []any{map[string]any{"device_type": "0x0510", "revision": 1}}},
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0001, Name: "ServerList", MatterType: "list", Mandatory: true, Origin: OriginDeclared, Declared: []any{"0x001D", "0x0090", "0x009C"}},
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0002, Name: "ClientList", MatterType: "list", Mandatory: true, Origin: OriginDeclared, Declared: []any{}},
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0003, Name: "PartsList", MatterType: "list", Mandatory: true, Origin: OriginDeclared, Declared: []any{}},
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Mandatory: true, Origin: OriginDeclared, Declared: 0},
	{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Mandatory: true, Origin: OriginDeclared, Declared: 3},
	{ClusterID: 0x009C, ClusterName: "Power Topology", AttributeID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Mandatory: true, Origin: OriginMissing, Detail: "the feature selection has no catalog element and no recorded design sentence; the specification does not fix it, so none is declared"},
	{ClusterID: 0x009C, ClusterName: "Power Topology", AttributeID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Mandatory: true, Origin: OriginDeclared, Declared: 1},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Mandatory: true, Origin: OriginMissing, Detail: "the feature selection has no catalog element and no recorded design sentence; the specification does not fix it, so none is declared"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Mandatory: true, Origin: OriginDeclared, Declared: 3},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0000, Name: "PowerMode", MatterType: "enum8", Mandatory: true, Origin: OriginMissing, Detail: "no catalog element states the current type of the measured subject; no value is derived from a fact prefix and none is declared"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0001, Name: "NumberOfMeasurementTypes", MatterType: "uint8", Mandatory: true, Origin: OriginMissing, Detail: "no catalog count and no recorded design sentence; it is not derived from a count of accuracy entries"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", MatterType: "list", Mandatory: true, Origin: OriginMissing, Detail: "no catalog element"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "MeasurementType", MatterType: "enum16", Mandatory: true, Origin: OriginMissing, Detail: "the catalog dimension has no roster row and no cited field list ties its symbols to the specification's measurement types; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "Measured", MatterType: "bool", Mandatory: true, Origin: OriginMissing, Detail: "pv.accuracy.measured has no cited field mapping; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "MinMeasuredValue", MatterType: "int64", Mandatory: true, Origin: OriginMissing, Detail: "the four catalog elements for active current and active power name no cited conversion bounds and no other measurement type has an element; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "MaxMeasuredValue", MatterType: "int64", Mandatory: true, Origin: OriginMissing, Detail: "the four catalog elements for active current and active power name no cited conversion bounds and no other measurement type has an element; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "AccuracyRanges", MatterType: "list", Mandatory: true, Origin: OriginMissing, Detail: "no catalog element"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "RangeMin", MatterType: "int64", Mandatory: true, Origin: OriginMissing, Detail: "the four catalog elements for active current and active power name no cited conversion bounds and no other measurement type has an element; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "RangeMax", MatterType: "int64", Mandatory: true, Origin: OriginMissing, Detail: "the four catalog elements for active current and active power name no cited conversion bounds and no other measurement type has an element; the row stays unknown"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0008, Name: "ActivePower", MatterType: "power-mW", Nullable: true, Mandatory: true, Origin: OriginMissing, Detail: "the catalog sign of evse.ac.active_power is the native convention and no cited fact ties it to the sign rule of the attribute; the attribute is nullable, so ActivePower is null for the pilot"},
	{ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0005, Name: "ActiveCurrent", MatterType: "amperage-mA", Nullable: true, Origin: OriginSemreg, PackID: "helianthus.pack.evse", PackVersion: "1.0.0", Element: "evse.ac.current", Dimension: "evse.dimension.phase", Encoder: "ampere_to_milliampere", Reason: "matter.phase_endpoint_identity_loss.v1"},
}
