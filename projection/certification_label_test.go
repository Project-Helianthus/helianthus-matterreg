package projection

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	epmdef "thingfabric.dev/go-fabric/cluster/spec/electricalpowermeasurement"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/fixedlabel"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

type vectorFact struct {
	Element     string `json:"element"`
	Type        string `json:"type"`
	Origin      string `json:"origin"`
	Kind        string `json:"kind"`
	Coefficient string `json:"coefficient"`
	Exponent10  int32  `json:"exponent10"`
	Unit        string `json:"unit"`
	Boolean     bool   `json:"boolean"`
}

type vectorEntry struct {
	Type     string   `json:"type"`
	Source   string   `json:"source"`
	Percent  *uint16  `json:"percent_max"`
	RangeMin *string  `json:"range_min"`
	RangeMax *string  `json:"range_max"`
	Measured *bool    `json:"measured"`
	MinMeas  *string  `json:"min_measured_value"`
	MaxMeas  *string  `json:"max_measured_value"`
	Missing  []string `json:"missing"`
}

type vector struct {
	ID     string       `json:"id"`
	Facts  []vectorFact `json:"facts"`
	Expect struct {
		Power    []vectorEntry      `json:"power"`
		Energy   *vectorEntry       `json:"energy"`
		Labels   []fixedlabel.Entry `json:"labels"`
		Refusals []struct {
			Element string `json:"element"`
			Type    string `json:"type"`
		} `json:"refusals"`
	} `json:"expect"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	raw, err := os.ReadFile("testdata/certification-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Vectors []vector `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Vectors
}

func accuracyEvidence(kind semreg.DefinitionID) semreg.EvidenceRef {
	e := matterEvidence()
	e.Kind = kind
	return e
}

func accuracyCandidate(i int, f vectorFact) (semreg.FactCandidate, semreg.EvaluatedFact) {
	inverter := "inverter-1"
	key := semreg.FactKey{
		PackID: "helianthus.pack.pv", PackVersion: "1.0.0", FactID: semreg.DefinitionID("pv." + f.Element),
		Dimensions: []semreg.Dimension{
			{ID: "pv.dimension.inverter", Value: semreg.Value{Kind: semreg.ValueText, Text: &inverter}},
			{ID: "pv.dimension.measurement_type", Value: semreg.Value{Kind: semreg.ValueSymbol, Symbol: &semreg.Symbol{Namespace: "pv.dimension.measurement_type", Token: f.Type, Known: true}}},
		},
	}
	var value semreg.Value
	if f.Kind == "boolean" {
		v := f.Boolean
		value = semreg.Value{Kind: semreg.ValueBoolean, Boolean: &v}
	} else {
		value = semreg.Value{Kind: semreg.ValueQuantity, Quantity: &semreg.Quantity{Number: semreg.Decimal{Coefficient: f.Coefficient, Exponent10: f.Exponent10}, Unit: semreg.DefinitionID(f.Unit)}}
	}
	id := semreg.CandidateID("candidate:accuracy:" + string(rune('a'+i)))
	candidate := semreg.FactCandidate{
		CandidateID: id, Key: key, Value: &value,
		Quality:  semreg.Quality{Assertion: semreg.AssertionObserved, Qualification: semreg.QualificationQualified, Promotion: semreg.PromotionPromoted, Validity: semreg.ValidityGood, Availability: semreg.AvailabilityAvailable, Freshness: semreg.FreshnessFresh, Reasons: []semreg.DefinitionID{}},
		Origin:   semreg.OriginRef{OriginID: "origin:accuracy", Kind: semreg.OriginNativeObservation, Evidence: []semreg.EvidenceRef{accuracyEvidence("evidence.test")}},
		Revision: "1",
	}
	if f.Origin == SourceCertification {
		candidate.Quality.Assertion = semreg.AssertionInferred
		candidate.Origin = semreg.OriginRef{OriginID: "origin:accuracy", Kind: semreg.OriginDeclared, Evidence: []semreg.EvidenceRef{accuracyEvidence(semreg.EvidenceKindCertification)}}
	}
	return candidate, semreg.EvaluatedFact{CandidateID: id, CandidateRevision: "1", Freshness: semreg.FreshnessFresh, EffectiveAvailability: semreg.AvailabilityAvailable}
}

func accuracySnapshot(facts []vectorFact) (semreg.Snapshot, semreg.EvaluationView) {
	var snapshot semreg.Snapshot
	var view semreg.EvaluationView
	for i, f := range facts {
		candidate, evaluated := accuracyCandidate(i, f)
		snapshot.Facts = append(snapshot.Facts, semreg.FactEnvelope{Key: candidate.Key, Candidates: []semreg.FactCandidate{candidate}, Conflicts: []semreg.Conflict{}, Revision: "1"})
		view.Facts = append(view.Facts, evaluated)
	}
	return snapshot, view
}

func entryMatches(t *testing.T, id string, got AccuracyEntry, want vectorEntry) {
	t.Helper()
	cluster := "Electrical Power Measurement"
	if got.ClusterID == electricalpowermeasurement.ElectricalEnergyMeasurementClusterID {
		cluster = "Electrical Energy Measurement"
	}
	var missing []string
	for _, name := range want.Missing {
		missing = append(missing, cluster+".Accuracy."+name)
	}
	if missing == nil {
		missing = []string{}
	}
	if got.MeasurementType != want.Type || got.Source != want.Source || !reflect.DeepEqual(got.Range.PercentMax, want.Percent) ||
		!reflect.DeepEqual(got.Range.RangeMin, want.RangeMin) || !reflect.DeepEqual(got.Range.RangeMax, want.RangeMax) ||
		!reflect.DeepEqual(got.Measured, want.Measured) || !reflect.DeepEqual(got.MinMeasuredValue, want.MinMeas) ||
		!reflect.DeepEqual(got.MaxMeasuredValue, want.MaxMeas) || !reflect.DeepEqual(got.Missing, missing) {
		t.Fatalf("%s: entry %+v, want %+v", id, got, want)
	}
}

func TestCertificationVectors(t *testing.T) {
	for _, v := range loadVectors(t) {
		t.Run(v.ID, func(t *testing.T) {
			snapshot, view := accuracySnapshot(v.Facts)
			got, err := projectAccuracy(snapshot, view)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Power) != len(v.Expect.Power) {
				t.Fatalf("power entries %+v, want %d", got.Power, len(v.Expect.Power))
			}
			for i := range got.Power {
				entryMatches(t, v.ID, got.Power[i], v.Expect.Power[i])
			}
			if (got.Energy == nil) != (v.Expect.Energy == nil) {
				t.Fatalf("energy entry %+v, want %+v", got.Energy, v.Expect.Energy)
			}
			if got.Energy != nil {
				entryMatches(t, v.ID, *got.Energy, *v.Expect.Energy)
			}
			wantLabels := v.Expect.Labels
			if len(wantLabels) == 0 {
				wantLabels = nil
			}
			if !reflect.DeepEqual(got.FixedLabel, wantLabels) {
				t.Fatalf("fixed label %+v, want %+v", got.FixedLabel, wantLabels)
			}
			if len(got.Refusals) != len(v.Expect.Refusals) {
				t.Fatalf("refusals %+v, want %+v", got.Refusals, v.Expect.Refusals)
			}
			for i, refusal := range got.Refusals {
				if refusal.Element != v.Expect.Refusals[i].Element || refusal.MeasurementType != v.Expect.Refusals[i].Type || refusal.Reason == "" {
					t.Fatalf("refusal %+v, want %+v", refusal, v.Expect.Refusals[i])
				}
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "fixed_max") {
				t.Fatalf("no catalog element states an absolute value, so FixedMax is never emitted: %s", encoded)
			}
		})
	}
}

func TestLabelsAreTheModelMemberNamesAndFit(t *testing.T) {
	for _, v := range loadVectors(t) {
		snapshot, view := accuracySnapshot(v.Facts)
		got, err := projectAccuracy(snapshot, view)
		if err != nil {
			t.Fatal(err)
		}
		members := map[string]bool{}
		for _, value := range epmdef.MeasurementTypeEnumDef.Values {
			members[value.Name] = true
		}
		for _, label := range got.FixedLabel {
			if !members[label.Label] || label.Value != fixedlabel.CertificationValue || fixedlabel.Check(label) != nil {
				t.Fatalf("%s: label %+v", v.ID, label)
			}
		}
	}
}

func TestMeasurementTypeRowsMatchTheModel(t *testing.T) {
	rows := electricalpowermeasurement.MeasurementTypes()
	if len(rows) != len(epmdef.MeasurementTypeEnumDef.Values) {
		t.Fatalf("%d rows against %d model members", len(rows), len(epmdef.MeasurementTypeEnumDef.Values))
	}
	for i, value := range epmdef.MeasurementTypeEnumDef.Values {
		if uint64(rows[i].ID) != uint64(value.Value) || rows[i].Name != value.Name {
			t.Fatalf("row %+v against model %+v", rows[i], value)
		}
	}
}

func TestPercentAndPercentMaximumMatchTheModel(t *testing.T) {
	for _, f := range epmdef.MeasurementAccuracyRangeStructDef.Fields {
		if f.Name == "PercentMax" && (!f.Constraint.Max.IsInt() || f.Constraint.Max.Int != electricalpowermeasurement.Percent100thsMaximum) {
			t.Fatalf("PercentMax bound %+v", f.Constraint.Max)
		}
	}
}

// Each case changes one field of what identifies a certification figure and
// the production check that notices it is named.
func TestACertificationFigureNeedsEveryPartOfItsIdentity(t *testing.T) {
	base := []vectorFact{{Element: "accuracy.ratio_maximum", Type: "active_power", Origin: SourceCertification, Kind: "quantity", Coefficient: "5", Exponent10: -3, Unit: "unit.ratio"}}
	cases := map[string]func(*semreg.Snapshot, *semreg.EvaluationView){
		"origin kind native instead of declared (accuracySource origin check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Origin.Kind = semreg.OriginNativeObservation
		},
		"origin evidence kind other than certification (evidence-kind check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Origin.Evidence = []semreg.EvidenceRef{accuracyEvidence("evidence.test")}
		},
		"qualification (quality check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Quality.Qualification = semreg.QualificationCandidate
		},
		"promotion (quality check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Quality.Promotion = semreg.PromotionUnpromoted
		},
		"validity (quality check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Quality.Validity = semreg.ValidityBad
		},
		"availability (quality check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Quality.Availability = semreg.AvailabilityUnavailable
		},
		"freshness of the evaluation (evaluationUsable check)": func(_ *semreg.Snapshot, v *semreg.EvaluationView) {
			v.Facts[0].Freshness = semreg.FreshnessStale
		},
		"availability of the evaluation (evaluationUsable check)": func(_ *semreg.Snapshot, v *semreg.EvaluationView) {
			v.Facts[0].EffectiveAvailability = semreg.AvailabilityUnavailable
		},
		"value absent (value check)": func(s *semreg.Snapshot, _ *semreg.EvaluationView) {
			s.Facts[0].Candidates[0].Value = nil
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			snapshot, view := accuracySnapshot(base)
			mutate(&snapshot, &view)
			got, err := projectAccuracy(snapshot, view)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Empty() || got.FixedLabel != nil {
				t.Fatalf("a figure without every part of its identity was used: %+v", got)
			}
		})
	}
}

func TestMeasurementTypeAndElementMismatchIsRefused(t *testing.T) {
	facts := []vectorFact{{Element: "accuracy.range_maximum_active_power", Type: "active_current", Origin: SourceCertification, Kind: "quantity", Coefficient: "1", Unit: "unit.watt"}}
	snapshot, view := accuracySnapshot(facts)
	got, err := projectAccuracy(snapshot, view)
	if err != nil || len(got.Power) != 0 || got.FixedLabel != nil || len(got.Refusals) != 1 {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestInvalidFactValueIsAnError(t *testing.T) {
	facts := []vectorFact{{Element: "accuracy.range_maximum_active_power", Type: "active_power", Origin: SourceCertification, Kind: "quantity", Coefficient: "1", Unit: "unit.ampere"}}
	snapshot, view := accuracySnapshot(facts)
	if _, err := projectAccuracy(snapshot, view); err == nil {
		t.Fatal("accepted a range in the wrong unit")
	}
}

func TestTwoFiguresForOneMemberAreAmbiguous(t *testing.T) {
	fact := vectorFact{Element: "accuracy.ratio_maximum", Type: "active_power", Origin: SourceCertification, Kind: "quantity", Coefficient: "5", Exponent10: -3, Unit: "unit.ratio"}
	snapshot, view := accuracySnapshot([]vectorFact{fact, fact})
	snapshot.Facts[1].Candidates[0].CandidateID = "candidate:accuracy:second"
	view.Facts[1].CandidateID = "candidate:accuracy:second"
	if _, err := projectAccuracy(snapshot, view); err == nil {
		t.Fatal("accepted two current certification figures for one member")
	}
}

func TestConflictIsAnError(t *testing.T) {
	snapshot, view := accuracySnapshot([]vectorFact{{Element: "accuracy.ratio_maximum", Type: "active_power", Origin: SourceCertification, Kind: "quantity", Coefficient: "5", Exponent10: -3, Unit: "unit.ratio"}})
	snapshot.Facts[0].Conflicts = []semreg.Conflict{{}}
	if _, err := projectAccuracy(snapshot, view); err == nil {
		t.Fatal("accepted a conflict")
	}
}

func TestSourceLessMembersStayMissingInTheProfile(t *testing.T) {
	source, err := loadPilotRows()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Power Topology.FeatureMap", "Electrical Power Measurement.FeatureMap", "Electrical Power Measurement.PowerMode",
		"Electrical Power Measurement.NumberOfMeasurementTypes",
	}
	for _, name := range want {
		if !source[name] {
			t.Fatalf("%s is not a missing row of the profile", name)
		}
	}
}

func loadPilotRows() (map[string]bool, error) {
	out := map[string]bool{}
	for _, row := range pilot.Rows() {
		if row.Mandatory && row.Origin == pilot.OriginMissing {
			out[row.ReportName()] = true
		}
	}
	return out, nil
}

func TestProfileProjectionCarriesTheDeviceAccuracyAndNoLabel(t *testing.T) {
	meter, phase := "meter-1", "l1"
	candidate := matterCandidate("candidate:matter:accuracy", phase, "1", -2)
	ratio := semreg.Value{Kind: semreg.ValueQuantity, Quantity: &semreg.Quantity{Number: semreg.Decimal{Coefficient: "5", Exponent10: -3}, Unit: "unit.ratio"}}
	candidate.Value = &ratio
	candidate.Key = semreg.FactKey{
		PackID: "helianthus.pack.evse", PackVersion: "1.0.0", FactID: "evse.accuracy.ratio_maximum",
		Dimensions: []semreg.Dimension{
			{ID: "evse.dimension.measurement_type", Value: semreg.Value{Kind: semreg.ValueSymbol, Symbol: &semreg.Symbol{Namespace: "evse.dimension.measurement_type", Token: "active_current", Known: true}}},
			{ID: "evse.dimension.meter", Value: semreg.Value{Kind: semreg.ValueText, Text: &meter}},
		},
	}
	fixture := newMatterFixture(t, []semreg.FactCandidate{candidate})
	result, err := ProjectProfile161(pilot.ResourceKind, fixture.snapshot, fixture.evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accuracy == nil || len(result.Accuracy.Power) != 1 || result.Accuracy.FixedLabel != nil {
		t.Fatalf("accuracy %+v", result.Accuracy)
	}
	entry := result.Accuracy.Power[0]
	if entry.MeasurementType != "ActiveCurrent" || entry.Source != SourceDevice || entry.Range.PercentMax == nil || *entry.Range.PercentMax != 50 {
		t.Fatalf("entry %+v", entry)
	}
	if result.Admitted {
		t.Fatal("a source-less mandatory member was completed")
	}
}
