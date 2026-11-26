package thermostat

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0201 || table.Revision != 11 || table.Name != "Thermostat" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]struct {
		bit         uint
		conformance string
	}{"HEAT": {0, "AUTO, O.a+"}, "COOL": {1, "AUTO, O.a+"}, "OCC": {2, "O"}, "SB": {4, "D"}, "AUTO": {5, "O"}, "LTNE": {6, "O"}, "MSCH": {7, "O"}, "PRES": {8, "O"}, "TEVT": {9, "P, O"}, "TSUGGEST": {10, "[PRES]"}}
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
	want := map[string]uint32{"LocalTemperature": 0x0, "ControlSequenceOfOperation": 0x1B, "SystemMode": 0x1C, "FeatureMap": 0xFFFC, "ClusterRevision": 0xFFFD}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	conditional := map[string]string{"Occupancy": "OCC", "OccupiedHeatingSetpoint": "HEAT", "OccupiedCoolingSetpoint": "COOL", "UnoccupiedHeatingSetpoint": "HEAT & OCC", "UnoccupiedCoolingSetpoint": "COOL & OCC", "MinHeatSetpointLimit": "[HEAT]", "MaxHeatSetpointLimit": "[HEAT]", "AbsMinHeatSetpointLimit": "[HEAT]", "AbsMaxHeatSetpointLimit": "[HEAT]"}
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
	wantCommands := map[uint32]struct{ name, conformance string }{0x00: {"SetpointRaiseLower", "M"}, 0x02: {"AddThermostatSuggestionResponse", "TSUGGEST"}, 0x05: {"SetActiveScheduleRequest", "MSCH"}, 0x06: {"SetActivePresetRequest", "PRES"}, 0x07: {"AddThermostatSuggestion", "TSUGGEST"}, 0x08: {"RemoveThermostatSuggestion", "TSUGGEST"}, 0xFD: {"AtomicResponse", ""}, 0xFE: {"AtomicRequest", "PRES | MSCH"}}
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
