package powertopology

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x009C || table.Revision != 1 || table.Name != "Power Topology" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]uint{"NODE": 0, "TREE": 1, "SET": 2, "DYPF": 3}
	if len(table.Features) != len(wantFeatures) {
		t.Fatalf("features %#v", table.Features)
	}
	for _, f := range table.Features {
		if bit, ok := wantFeatures[f.Code]; !ok || bit != f.Bit {
			t.Fatalf("feature %#v", f)
		}
	}
	mandatory := map[string]uint32{}
	for _, a := range table.Attributes {
		if a.Mandatory {
			mandatory[a.Name] = a.ID
		}
	}
	want := map[string]uint32{"FeatureMap": 0xfffc, "ClusterRevision": 0xfffd}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	wantCommands := []string{}
	if len(table.Commands) != len(wantCommands) {
		t.Fatalf("commands %#v", table.Commands)
	}
	for i, c := range table.Commands {
		if c.ID != uint32(i) || c.Name != wantCommands[i] {
			t.Fatalf("command %#v", c)
		}
	}
	if len(table.Mappings) != 0 {
		t.Fatalf("the target maps no attribute of this cluster: %#v", table.Mappings)
	}
}
