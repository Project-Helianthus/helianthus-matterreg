package identify

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0003 || table.Revision != 6 || table.Name != "Identify" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]struct {
		bit         uint
		conformance string
	}{}
	if len(table.Features) != len(wantFeatures) {
		t.Fatalf("features %#v", table.Features)
	}
	for _, f := range table.Features {
		if w, ok := wantFeatures[f.Code]; !ok || w.bit != f.Bit || w.conformance != f.Conformance {
			t.Fatalf("feature %#v", f)
		}
	}
	mandatory := map[string]uint32{}
	for _, a := range table.Attributes {
		if a.Mandatory {
			mandatory[a.Name] = a.ID
		}
	}
	want := map[string]uint32{"IdentifyTime": 0x0, "IdentifyType": 0x1, "FeatureMap": 0xFFFC, "ClusterRevision": 0xFFFD}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	conditional := map[string]string{}
	for _, a := range table.Attributes {
		if c, ok := conditional[a.Name]; ok {
			if a.Mandatory || a.Conformance != c {
				t.Fatalf("attribute %#v", a)
			}
			delete(conditional, a.Name)
		}
	}
	if len(conditional) != 0 {
		t.Fatalf("attributes missing from the table %v", conditional)
	}
	wantCommands := map[uint32]struct{ name, conformance string }{0x00: {"Identify", "M"}, 0x40: {"TriggerEffect", "O"}}
	if len(table.Commands) != len(wantCommands) {
		t.Fatalf("commands %#v", table.Commands)
	}
	for _, c := range table.Commands {
		if w, ok := wantCommands[c.ID]; !ok || w.name != c.Name || w.conformance != c.Conformance {
			t.Fatalf("command %#v", c)
		}
	}
	if len(table.Mappings) != 0 {
		t.Fatalf("the target maps no attribute of this cluster: %#v", table.Mappings)
	}
}
