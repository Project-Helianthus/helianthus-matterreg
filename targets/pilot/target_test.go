package pilot

import (
	"os"
	"strings"
	"testing"
)

func syntheticTarget(t *testing.T) Target {
	t.Helper()
	raw, err := os.ReadFile("testdata/synthetic-target.json")
	if err != nil {
		t.Fatal(err)
	}
	target, err := LoadTarget(raw)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func missingNames(t *testing.T) []string {
	t.Helper()
	var names []string
	for _, row := range Rows() {
		if row.Origin == OriginMissing && row.Mandatory {
			names = append(names, row.ReportName())
		}
	}
	return names
}

func TestCommittedTargetRefusesAndNamesEveryGap(t *testing.T) {
	decision := Evaluate(CommittedTarget())
	if decision.Permitted || decision.Rows != nil {
		t.Fatalf("committed target composed an endpoint: %#v", decision)
	}
	want := missingNames(t)
	if len(want) != 13 || strings.Join(decision.Missing, "|") != strings.Join(want, "|") {
		t.Fatalf("missing %v", decision.Missing)
	}
	for i, row := range decision.Reasons {
		if row == "" || row == noSourceReason {
			t.Fatalf("gap %s has no target reason", decision.Missing[i])
		}
	}
	for _, row := range Rows() {
		if row.Origin == OriginMissing && (row.Element != "" || row.Declared != nil || row.Encoder != "") {
			t.Fatalf("gap %s holds a value", row.ReportName())
		}
	}
}

func TestSyntheticTargetAdmitsCompleteEndpoint(t *testing.T) {
	decision := Evaluate(syntheticTarget(t))
	if !decision.Permitted || len(decision.Missing) != 0 || decision.Rows == nil {
		t.Fatalf("decision %#v", decision)
	}
	for _, row := range decision.Rows {
		if row.Origin == OriginMissing {
			t.Fatalf("row %s still missing", row.ReportName())
		}
		if row.Origin == OriginSemreg && (row.Element == "" || row.Encoder == "") {
			t.Fatalf("row %s has no source", row.ReportName())
		}
	}
}

// Each vector removes the source of one mandatory value. Mutation caught: a
// default or another row's source standing in for the removed one.
func TestEachRemovedSourceNamesItsRow(t *testing.T) {
	full := syntheticTarget(t)
	names := missingNames(t)
	for i, name := range names {
		var row AttributeRow
		for _, candidate := range Rows() {
			if candidate.Origin == OriginMissing && candidate.Mandatory && candidate.ReportName() == name {
				row = candidate
			}
		}
		reduced := Target{}
		for _, source := range full.Sources {
			if source.ClusterID == row.ClusterID && source.AttributeID == row.AttributeID && source.Field == row.Field {
				continue
			}
			reduced.Sources = append(reduced.Sources, source)
		}
		if len(reduced.Sources) != len(full.Sources)-1 {
			t.Fatalf("%d: %s selects %d sources", i, name, len(full.Sources)-len(reduced.Sources))
		}
		decision := Evaluate(reduced)
		if decision.Permitted || decision.Rows != nil || len(decision.Missing) != 1 || decision.Missing[0] != name {
			t.Fatalf("%s: %#v", name, decision)
		}
		if decision.Reasons[0] != row.Detail {
			t.Fatalf("%s reason %q", name, decision.Reasons[0])
		}
	}
}

func TestSyntheticTargetMustSayItIsSynthetic(t *testing.T) {
	for _, raw := range []string{
		`{"sources":[]}`,
		`{"synthetic_test_data":"","sources":[]}`,
		`{"sources":[],"synthetic_test_data":"x"}`,
		`{"synthetic_test_data":"x","sources":[{"cluster_id":144}]}`,
	} {
		if _, err := LoadTarget([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
