package energyevse

import "testing"

func TestTableFacts(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0099 || table.Revision != 4 || table.Name != "Energy EVSE" {
		t.Fatalf("identity %#v", table)
	}
	wantFeatures := map[string]struct {
		bit         uint
		conformance string
	}{"PREF": {0, "M"}, "SOC": {1, "O"}, "PNC": {2, "O"}, "RFID": {3, "O"}, "V2X": {4, "O"}}
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
	want := map[string]uint32{
		"State": 0x0, "SupplyState": 0x1, "FaultState": 0x2, "ChargingEnabledUntil": 0x3, "CircuitCapacity": 0x5,
		"MinimumChargeCurrent": 0x6, "MaximumChargeCurrent": 0x7, "SessionId": 0x40, "SessionDuration": 0x41,
		"SessionEnergyCharged": 0x42, "FeatureMap": 0xfffc, "ClusterRevision": 0xfffd,
	}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if got, ok := mandatory[name]; !ok || got != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	conditional := map[string]string{
		"DischargingEnabledUntil": "V2X", "MaximumDischargeCurrent": "V2X", "UserMaximumChargeCurrent": "O",
		"RandomizationDelayWindow": "O", "NextChargeStartTime": "PREF", "NextChargeTargetTime": "PREF",
		"NextChargeRequiredEnergy": "PREF", "NextChargeTargetSoC": "PREF", "ApproximateEvEfficiency": "[PREF]",
		"StateOfCharge": "SOC", "BatteryCapacity": "SOC", "VehicleId": "PNC", "SessionEnergyDischarged": "V2X",
	}
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
	wantCommands := map[uint32]struct{ name, conformance string }{
		0x00: {"GetTargetsResponse", "PREF"}, 0x01: {"Disable", "M"}, 0x02: {"EnableCharging", "M"},
		0x03: {"EnableDischarging", "V2X"}, 0x04: {"StartDiagnostics", "O"}, 0x05: {"SetTargets", "PREF"},
		0x06: {"GetTargets", "PREF"}, 0x07: {"ClearTargets", "PREF"},
	}
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
