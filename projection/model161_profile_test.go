package projection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/projection"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

func TestProfileProjectsEncodedCurrentAndDeclaredRows(t *testing.T) {
	f := aa03Fixture(t)
	got := projectProfile(t, f)
	if got.Admitted {
		t.Fatal("endpoint admitted while mandatory rows are missing")
	}
	if !reflect.DeepEqual(got.Missing, profileMissingRows()) {
		t.Fatalf("missing %#v", got.Missing)
	}
	if len(got.Attributes) < 2 {
		t.Fatalf("attributes %#v", got.Attributes)
	}
	current := profileAttribute(t, got, "ActiveCurrent")
	if current.Null || current.Value != "3200" || current.Element != "evse.ac.current" || current.Encoder != "ampere_to_milliampere" {
		t.Fatalf("current %#v", current)
	}
	copied := aa03Positive(t, f)
	if copied.Document.Attributes[0].Value.Milliampere != "3200" || len(copied.Document.Attributes) != 1 {
		t.Fatal("copied ActiveCurrent changed")
	}
	device := profileAttribute(t, got, "DeviceTypeList")
	entry := device.Value.([]any)[0].(map[string]any)
	if !device.Declared || entry["device_type"] != "0x0510" || entry["revision"] != 1 {
		t.Fatalf("device type %#v", device.Value)
	}
	if len(got.Folds) != 1 || !got.Folds[0].Declared || got.Folds[0].Kind != "identity" {
		t.Fatalf("folds %#v", got.Folds)
	}
	disposition := got.Dispositions[0]
	if disposition.Outcome != projection.ProjectionTransformed || disposition.Reason == nil || *disposition.Reason != "matter.phase_endpoint_identity_loss.v1" {
		t.Fatalf("disposition %#v", disposition)
	}
	if len(disposition.Loss) != 2 || disposition.Loss[0].Kind != projection.LossIdentity || disposition.Loss[1].Kind != projection.LossUnit {
		t.Fatalf("loss %#v", disposition.Loss)
	}
	scaled := newMatterFixture(t, []semreg.FactCandidate{matterCandidate("candidate:profile:scaled", "l1", "1234", -3)})
	scaledResult := projectProfile(t, scaled)
	if profileAttribute(t, scaledResult, "ActiveCurrent").Value != "1234" {
		t.Fatalf("scaled %#v", profileAttribute(t, scaledResult, "ActiveCurrent"))
	}
	phase := newMatterFixture(t, []semreg.FactCandidate{aa03Candidate("l2")})
	if profileAttribute(t, projectProfile(t, phase), "ActiveCurrent").Value != "3200" {
		t.Fatal("phase value was treated as a different measurement")
	}
}

func TestProfileAbsentCurrentIsNull(t *testing.T) {
	f := newMatterFixture(t, []semreg.FactCandidate{})
	got := projectProfile(t, f)
	current := profileAttribute(t, got, "ActiveCurrent")
	if !current.Null || current.Value != nil {
		t.Fatalf("absent %#v", current)
	}
	raw, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(`"value":0`)) || bytes.Contains(raw, []byte(`"value":"0"`)) {
		t.Fatalf("absent current became zero: %s", raw)
	}
	if got.Dispositions[0].Outcome != projection.ProjectionUnknown || got.Dispositions[0].Reason == nil || *got.Dispositions[0].Reason != "matter.unmapped.v1" {
		t.Fatalf("disposition %#v", got.Dispositions[0])
	}
	if len(got.Folds) != 1 || !got.Folds[0].Declared {
		t.Fatal("fold dropped")
	}
	stale := aa03Fixture(t)
	stale.snapshot.Facts[0].Candidates[0].Quality.Freshness = semreg.FreshnessStale
	aa03Reseal(t, &stale)
	staleResult := projectProfile(t, stale)
	if !profileAttribute(t, staleResult, "ActiveCurrent").Null || profileAttribute(t, staleResult, "ActiveCurrent").Value != nil {
		t.Fatal("stale current became a value")
	}
}

func TestProfilePresentZeroStaysZero(t *testing.T) {
	f := newMatterFixture(t, []semreg.FactCandidate{matterCandidate("candidate:profile:zero", "l1", "0", 0)})
	current := profileAttribute(t, projectProfile(t, f), "ActiveCurrent")
	if current.Null || current.Value != "0" {
		t.Fatalf("present zero %#v", current)
	}
}

func TestProfileRefusesNonrepresentableAndAmbiguous(t *testing.T) {
	f := aa03Fixture(t)
	f.snapshot.Facts[0].Candidates[0].Value.Quantity.Number = semreg.Decimal{Coefficient: "1", Exponent10: -4}
	aa03Reseal(t, &f)
	got, err := ProjectProfile161(pilot.ResourceKind, f.snapshot, f.evaluation)
	if err == nil || !reflect.DeepEqual(got, ProfileResult{}) {
		t.Fatalf("nonrepresentable result %#v err %v", got, err)
	}
	ambiguous := newMatterFixture(t, []semreg.FactCandidate{aa03Candidate("l1"), matterCandidate("candidate:synthetic:second", "l2", "33", -1)})
	got, err = ProjectProfile161(pilot.ResourceKind, ambiguous.snapshot, ambiguous.evaluation)
	if err == nil || !reflect.DeepEqual(got, ProfileResult{}) {
		t.Fatalf("ambiguous result %#v err %v", got, err)
	}
}

func TestProfileKeyFieldsDoNotFillActiveCurrent(t *testing.T) {
	// FactKey fields are PackID, PackVersion, FactID and Dimensions.
	// Each mutation must not still encode the original 3200 mA.
	other := newMatterFixture(t, []semreg.FactCandidate{limitCurrentCandidate()})
	otherResult := projectProfile(t, other)
	if profileAttribute(t, otherResult, "ActiveCurrent").Value == "3200" || !profileAttribute(t, otherResult, "ActiveCurrent").Null {
		t.Fatal("FactID evse.limit.actual_current filled ActiveCurrent")
	}
	cases := map[string]func(*semreg.FactKey){
		"PackID":      func(key *semreg.FactKey) { key.PackID = "other.pack" },
		"PackVersion": func(key *semreg.FactKey) { key.PackVersion = "9.0.0" },
		"FactID":      func(key *semreg.FactKey) { key.FactID = "evse.ac.voltage" },
		"Dimensions":  func(key *semreg.FactKey) { key.Dimensions[0].ID = "evse.dimension.connector" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := aa03Fixture(t)
			mutate(&f.snapshot.Facts[0].Key)
			mutate(&f.snapshot.Facts[0].Candidates[0].Key)
			aa03Reseal(t, &f)
			got, err := ProjectProfile161(pilot.ResourceKind, f.snapshot, f.evaluation)
			if err != nil {
				if !reflect.DeepEqual(got, ProfileResult{}) {
					t.Fatalf("invalid %s still projected %#v", name, got)
				}
				return
			}
			if profileAttribute(t, got, "ActiveCurrent").Value == "3200" {
				t.Fatalf("%s still encoded 3200", name)
			}
		})
	}
}

func TestProfileUnknownKindHasNoEndpoint(t *testing.T) {
	f := newMatterFixture(t, []semreg.FactCandidate{})
	got, err := ProjectProfile161("resource.other", f.snapshot, f.evaluation)
	if err == nil || !reflect.DeepEqual(got, ProfileResult{}) {
		t.Fatalf("unknown kind %#v err %v", got, err)
	}
}

func TestLockMappingMatchesContractDigest(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "sdk", "matter-sdk.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Mapping struct {
			Path   string `json:"path"`
			Sha256 string `json:"sha256"`
		} `json:"mapping"`
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.Mapping.Path != DocsSemanticContractPath161 || lock.Mapping.Sha256 != DocsSemanticContractSHA256161 {
		t.Fatal("lock mapping is not the 1.6.1 contract pin")
	}
	source, err := composition.Load()
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal("pilot missing")
	}
	var document map[string]any
	if err := json.Unmarshal(profile.Document, &document); err != nil {
		t.Fatal(err)
	}
	mapping := document["mapping"].(map[string]any)
	if mapping["path"] != DocsSemanticContractPath161 || mapping["sha256"] != DocsSemanticContractSHA256161 {
		t.Fatal("profile mapping is not the 1.6.1 contract pin")
	}
}

func projectProfile(t *testing.T, f matterFixture) ProfileResult {
	t.Helper()
	got, err := ProjectProfile161(pilot.ResourceKind, f.snapshot, f.evaluation)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func profileAttribute(t *testing.T, result ProfileResult, name string) ProjectedAttribute {
	t.Helper()
	for _, attribute := range result.Attributes {
		if attribute.Name == name && attribute.MatterType != "" {
			return attribute
		}
	}
	t.Fatalf("attribute %s missing in %#v", name, result.Attributes)
	return ProjectedAttribute{}
}

func profileMissingRows() []string {
	return []string{
		"Power Topology.FeatureMap",
		"Electrical Power Measurement.FeatureMap",
		"Electrical Power Measurement.PowerMode",
		"Electrical Power Measurement.NumberOfMeasurementTypes",
		"Electrical Power Measurement.Accuracy",
		"Electrical Power Measurement.Accuracy.MeasurementType",
		"Electrical Power Measurement.Accuracy.Measured",
		"Electrical Power Measurement.Accuracy.MinMeasuredValue",
		"Electrical Power Measurement.Accuracy.MaxMeasuredValue",
		"Electrical Power Measurement.Accuracy.AccuracyRanges",
		"Electrical Power Measurement.Accuracy.RangeMin",
		"Electrical Power Measurement.Accuracy.RangeMax",
		"Electrical Power Measurement.ActivePower",
	}
}

func limitCurrentCandidate() semreg.FactCandidate {
	candidate := matterCandidate("candidate:profile:limit", "connector-1", "32", -1)
	candidate.Key.FactID = "evse.limit.actual_current"
	candidate.Key.Dimensions[0].ID = "evse.dimension.connector"
	return candidate
}
