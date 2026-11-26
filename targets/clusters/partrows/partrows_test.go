package partrows

import (
	"reflect"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

func table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x1234, Name: "Synthetic", Revision: 7,
		Features: []clusters.Feature{{Bit: 0, Code: "AAA"}, {Bit: 3, Code: "BBB"}},
		Attributes: []clusters.Attribute{
			{ID: 0, Name: "Always", Conformance: "M", Mandatory: true},
			{ID: 1, Name: "WithAAA", Conformance: "AAA"},
			{ID: 2, Name: "WithBBB", Conformance: "BBB"},
			{ID: 3, Name: "Named", Conformance: "O"},
			{ID: 4, Name: "Optional", Conformance: "O"},
			{ID: 5, Name: "Accuracy", Conformance: "M", Mandatory: true},
			{ID: 0xFFFC, Name: "FeatureMap", Conformance: "M", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", Conformance: "M", Mandatory: true},
		},
	}
}

func byName(rows []composition.AttributeRow) map[string]composition.AttributeRow {
	out := map[string]composition.AttributeRow{}
	for _, r := range rows {
		out[r.ReportName()] = r
	}
	return out
}

func TestMandatorySelection(t *testing.T) {
	c := Cluster{Table: table(), Features: []string{"BBB"}, Attributes: []string{"Named"}}
	rows := byName(c.Rows("p"))
	for _, name := range []string{"Synthetic.Always", "Synthetic.WithBBB", "Synthetic.Named", "Synthetic.Accuracy", "Synthetic.FeatureMap", "Synthetic.ClusterRevision"} {
		if _, ok := rows[name]; !ok {
			t.Fatalf("%s has no row", name)
		}
	}
	for _, name := range []string{"Synthetic.WithAAA", "Synthetic.Optional"} {
		if _, ok := rows[name]; ok {
			t.Fatalf("%s is not mandatory for this mounting", name)
		}
	}
}

func TestSourcesAndFeatureMap(t *testing.T) {
	c := Cluster{Table: table(), Features: []string{"AAA", "BBB"}}
	rows := byName(c.Rows("p"))
	if r := rows["Synthetic.FeatureMap"]; r.Origin != composition.OriginDeclared || r.Declared != 9 {
		t.Fatalf("feature map %#v", r)
	}
	if r := rows["Synthetic.ClusterRevision"]; r.Origin != composition.OriginDeclared || r.Declared != 7 {
		t.Fatalf("revision %#v", r)
	}
	if r := rows["Synthetic.Always"]; r.Origin != composition.OriginMissing || r.Reason != UnmappedReason || r.Declared != nil || r.PartKey != "p" {
		t.Fatalf("dynamic row %#v", r)
	}
	open := Cluster{Table: table(), Features: []string{"AAA"}, FeatureChoice: "open choice"}
	if r := byName(open.Rows("p"))["Synthetic.FeatureMap"]; r.Origin != composition.OriginMissing || r.Detail != "open choice" || r.Declared != nil {
		t.Fatalf("open feature map %#v", r)
	}
}

func TestAccuracyFields(t *testing.T) {
	var fields []string
	for _, r := range (Cluster{Table: table()}).Rows("p") {
		if r.Name == "Accuracy" && r.Field != "" {
			if r.Origin != composition.OriginMissing {
				t.Fatalf("%s origin %s", r.Field, r.Origin)
			}
			fields = append(fields, r.Field)
		}
	}
	want := []string{"MeasurementType", "Measured", "MinMeasuredValue", "MaxMeasuredValue", "AccuracyRanges", "RangeMin", "RangeMax"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("fields %v", fields)
	}
}

func TestDescriptor(t *testing.T) {
	rows := byName(Descriptor("p", 0x0011, 1, []uint32{0x001D, 0x002F}, true, nil))
	if got := rows["Descriptor.DeviceTypeList"].Declared; !reflect.DeepEqual(got, []any{map[string]any{"device_type": "0x0011", "revision": 1}}) {
		t.Fatalf("device type list %v", got)
	}
	if got := rows["Descriptor.ServerList"].Declared; !reflect.DeepEqual(got, []any{"0x001D", "0x002F"}) {
		t.Fatalf("server list %v", got)
	}
	if rows["Descriptor.FeatureMap"].Declared != 1 {
		t.Fatalf("feature map %v", rows["Descriptor.FeatureMap"].Declared)
	}
	if r := rows["Descriptor.TagList"]; r.Origin != composition.OriginMissing || r.Reason != UnmappedReason {
		t.Fatalf("tag list %#v", r)
	}
	plain := byName(Descriptor("", 0x0017, 1, []uint32{0x001D}, false, []string{"a", "b"}))
	if _, ok := plain["Descriptor.TagList"]; ok {
		t.Fatal("tag list without TAGLIST")
	}
	if plain["Descriptor.FeatureMap"].Declared != 0 || !reflect.DeepEqual(plain["Descriptor.PartsList"].Declared, []any{"a", "b"}) {
		t.Fatalf("plain %#v", plain)
	}
}
