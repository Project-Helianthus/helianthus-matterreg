package electricalpowermeasurement

import (
	"errors"
	"testing"
)

func TestAmpereToMilliampereVectors(t *testing.T) {
	cases := []struct {
		name        string
		coefficient string
		exponent    int
		want        string
		err         error
	}{
		{"whole amperes", "5", 0, "5000", nil},
		{"two and a quarter", "225", -2, "2250", nil},
		{"negative", "-15", -1, "-1500", nil},
		{"zero", "0", 0, "0", nil},
		{"positive exponent", "5", 3, "5000000", nil},
		{"exactly one milliampere", "1", -3, "1", nil},
		{"half a milliampere is reported", "5", -4, "", ErrNotIntegral},
		{"third of a milliampere is reported", "1", -7, "", ErrNotIntegral},
		{"upper bound", "4611686018427387904", -3, "4611686018427387904", nil},
		{"above upper bound", "4611686018427387905", -3, "", ErrRange},
		{"lower bound", "-4611686018427387904", -3, "-4611686018427387904", nil},
		{"below lower bound", "-4611686018427387905", -3, "", ErrRange},
		{"not a number", "1.5", 0, "", ErrInput},
		{"empty", "", 0, "", ErrInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AmpereToMilliampere(tc.coefficient, tc.exponent)
			if !errors.Is(err, tc.err) || got != tc.want {
				t.Fatalf("got %q %v, want %q %v", got, err, tc.want, tc.err)
			}
		})
	}
}

func TestTableIdentityAndMapping(t *testing.T) {
	table := Table()
	if table.ClusterID != 0x0090 || table.Revision != 3 || table.Name != "Electrical Power Measurement" {
		t.Fatalf("identity %#v", table)
	}
	if len(table.Commands) != 0 {
		t.Fatal("the cluster has no command")
	}
	wantFeatures := map[string]uint{"DIRC": 0, "ALTC": 1, "POLY": 2, "HARM": 3, "PWRQ": 4}
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
	want := map[string]uint32{"PowerMode": 0, "NumberOfMeasurementTypes": 1, "Accuracy": 2, "ActivePower": 8, "FeatureMap": 0xFFFC, "ClusterRevision": 0xFFFD}
	if len(mandatory) != len(want) {
		t.Fatalf("mandatory %v", mandatory)
	}
	for name, id := range want {
		if mandatory[name] != id {
			t.Fatalf("mandatory %s %v", name, mandatory)
		}
	}
	if len(table.Mappings) != 1 {
		t.Fatalf("mappings %#v", table.Mappings)
	}
	m := table.Mappings[0]
	if m.AttributeID != 0x0005 || m.Element != "evse.ac.current" || m.Factor != 1000 || m.Offset != 0 || m.Rounding != "exact" || !m.RequireIntegral {
		t.Fatalf("mapping %#v", m)
	}
	for _, a := range table.Attributes {
		if a.ID == 0x0005 && (a.Name != "ActiveCurrent" || a.MatterType != "amperage-mA" || !a.Nullable || a.Mandatory) {
			t.Fatalf("ActiveCurrent %#v", a)
		}
	}
}
