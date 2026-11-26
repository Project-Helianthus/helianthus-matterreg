package energyevsemode

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x009D || table.Revision != 2 || table.Name != "Energy EVSE Mode" {
		t.Fatalf("identity %#v", table)
	}
	if len(table.Features) != 1 || table.Features[0].Code != "DEPONOFF" || table.Features[0].Bit != 0 || table.Features[0].Conformance != "X" {
		t.Fatalf("features %#v", table.Features)
	}
	mandatory := map[string]uint32{}
	for _, a := range table.Attributes {
		if a.Mandatory {
			mandatory[a.Name] = a.ID
		}
	}
	want := map[string]uint32{"SupportedModes": 0x0, "CurrentMode": 0x1, "FeatureMap": 0xfffc, "ClusterRevision": 0xfffd}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	for _, a := range table.Attributes {
		if (a.Name == "StartUpMode" || a.Name == "OnMode") && (a.Mandatory || a.Conformance != "X" || !a.Nullable) {
			t.Fatalf("attribute %#v", a)
		}
	}
	if len(table.Commands) != 2 || table.Commands[0].ID != 0 || table.Commands[0].Name != "ChangeToMode" ||
		table.Commands[1].ID != 1 || table.Commands[1].Name != "ChangeToModeResponse" {
		t.Fatalf("commands %#v", table.Commands)
	}
	if len(table.Mappings) != 0 {
		t.Fatalf("the target maps no attribute of this cluster: %#v", table.Mappings)
	}
}
