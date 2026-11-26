package deviceenergymanagement

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0098 || table.Revision != 4 || table.Name != "Device Energy Management" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]uint{"PA": 0, "PFR": 1, "SFR": 2, "STA": 3, "PAU": 4, "FA": 5, "CON": 6}
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
	want := map[string]uint32{"EsaType": 0x0, "EsaCanGenerate": 0x1, "EsaState": 0x2, "AbsMinPower": 0x3, "AbsMaxPower": 0x4, "FeatureMap": 0xfffc, "ClusterRevision": 0xfffd}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	wantCommands := []string{"PowerAdjustRequest", "CancelPowerAdjustRequest", "StartTimeAdjustRequest", "PauseRequest", "ResumeRequest", "ModifyForecastRequest", "RequestConstraintBasedForecast", "CancelRequest"}
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
