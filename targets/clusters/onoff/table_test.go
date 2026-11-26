package onoff

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0006 || table.Revision != 6 || table.Name != "On/Off" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]struct {
		bit         uint
		conformance string
	}{"LT": {0, "[!OFFONLY]"}, "DF": {1, "[!OFFONLY]"}, "OFFONLY": {2, "[!(LT | DF)]"}}
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
	want := map[string]uint32{"OnOff": 0x0, "FeatureMap": 0xFFFC, "ClusterRevision": 0xFFFD}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	conditional := map[string]string{"GlobalSceneControl": "LT", "OnTime": "LT", "OffWaitTime": "LT", "StartUpOnOff": "LT"}
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
	wantCommands := map[uint32]struct{ name, conformance string }{0x00: {"Off", "M"}, 0x01: {"On", "!OFFONLY"}, 0x02: {"Toggle", "!OFFONLY"}, 0x40: {"OffWithEffect", "LT"}, 0x41: {"OnWithRecallGlobalScene", "LT"}, 0x42: {"OnWithTimedOff", "LT"}}
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
