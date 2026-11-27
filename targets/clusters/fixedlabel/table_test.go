package fixedlabel

import (
	"errors"
	"strings"
	"testing"

	epmdef "thingfabric.dev/go-fabric/cluster/spec/electricalpowermeasurement"
	fixeddef "thingfabric.dev/go-fabric/cluster/spec/fixedlabel"
)

func TestTableMatchesModel(t *testing.T) {
	table := Table()
	if table.ClusterID != fixeddef.ClusterID || table.Revision != fixeddef.Revision {
		t.Fatalf("identity %#v against the model", table)
	}
	found := false
	for _, a := range table.Attributes {
		if a.Name == "LabelList" {
			found = true
			if a.ID != fixeddef.AttrLabelList || !a.Mandatory || a.Nullable {
				t.Fatalf("LabelList row %#v", a)
			}
		}
	}
	if !found {
		t.Fatal("LabelList row missing")
	}
	if len(table.Features) != 0 || len(table.Commands) != 0 || len(table.Mappings) != 0 {
		t.Fatalf("the cluster has no feature, command or mapping: %#v", table)
	}
}

func TestMaximumMatchesModel(t *testing.T) {
	for _, field := range fixeddef.LabelStructDef.Fields {
		if !field.Constraint.Max.IsInt() || field.Constraint.Max.Int != MaxLength {
			t.Fatalf("model maximum of %s differs from %d", field.Name, MaxLength)
		}
	}
}

func TestCheckRefusesEveryField(t *testing.T) {
	long := strings.Repeat("x", MaxLength+1)
	exact := strings.Repeat("x", MaxLength)
	cases := []struct {
		name  string
		entry Entry
		err   error
	}{
		{"both at the maximum", Entry{Label: exact, Value: exact}, nil},
		{"label over", Entry{Label: long, Value: CertificationValue}, ErrLength},
		{"value over", Entry{Label: "ActivePower", Value: long}, ErrLength},
	}
	for _, tc := range cases {
		if err := Check(tc.entry); !errors.Is(err, tc.err) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestEveryMeasurementTypeNameFits(t *testing.T) {
	for _, v := range epmdef.MeasurementTypeEnumDef.Values {
		if err := Check(Entry{Label: v.Name, Value: CertificationValue}); err != nil {
			t.Fatalf("%s: %v", v.Name, err)
		}
	}
}
