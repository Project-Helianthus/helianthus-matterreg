package pumpconfigurationandcontrol

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0200 || table.Revision != 5 || table.Name != "Pump Configuration and Control" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]struct {
		bit         uint
		conformance string
	}{"PRSCONST": {0, "O.a+"}, "PRSCOMP": {1, "O.a+"}, "FLW": {2, "O.a+"}, "SPD": {3, "O.a+"}, "TEMP": {4, "O.a+"}, "AUTO": {5, "O"}, "LOCAL": {6, "O"}}
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
	want := map[string]uint32{"MaxPressure": 0x0, "MaxSpeed": 0x1, "MaxFlow": 0x2, "EffectiveOperationMode": 0x11, "EffectiveControlMode": 0x12, "Capacity": 0x13, "OperationMode": 0x20, "FeatureMap": 0xFFFC, "ClusterRevision": 0xFFFD}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	conditional := map[string]string{"MinConstPressure": "PRSCONST, [AUTO]", "MinConstSpeed": "SPD, [AUTO]", "MinConstFlow": "FLW, [AUTO]", "MinConstTemp": "TEMP, [AUTO]", "PumpStatus": "O"}
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
	wantCommands := map[uint32]struct{ name, conformance string }{}
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
