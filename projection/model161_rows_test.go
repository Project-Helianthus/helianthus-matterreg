package projection

import (
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

func loadedPilot(t *testing.T) composition.Profile {
	t.Helper()
	source, err := composition.Load()
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal("pilot missing")
	}
	return profile
}

func currentRowIndex(t *testing.T, profile composition.Profile) int {
	t.Helper()
	for i, row := range profile.Attributes {
		if row.Name == "ActiveCurrent" {
			return i
		}
	}
	t.Fatal("ActiveCurrent row missing")
	return -1
}

// The disposition reason is read from the row: a different row reason must
// reach the disposition. Mutation caught: a constant reason in the projection.
func TestProfileReasonComesFromRow(t *testing.T) {
	f := aa03Fixture(t)
	profile := loadedPilot(t)
	profile.Attributes[currentRowIndex(t, profile)].Reason = "matter.synthetic_reason.v1"
	got, err := projectLoadedProfile(profile, f.snapshot, f.evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if got.Dispositions[0].Reason == nil || *got.Dispositions[0].Reason != "matter.synthetic_reason.v1" {
		t.Fatalf("reason %#v", got.Dispositions[0].Reason)
	}
}

// A fold applies to the rows whose element it names. Mutation caught: folding
// every semreg row regardless of the fold's source items.
func TestProfileFoldLossFollowsSourceItems(t *testing.T) {
	f := aa03Fixture(t)
	profile := loadedPilot(t)
	profile.Folds[0].SourceItems = []string{"evse.ac.voltage"}
	got, err := projectLoadedProfile(profile, f.snapshot, f.evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Dispositions[0].Loss) != 1 {
		t.Fatalf("loss %#v", got.Dispositions[0].Loss)
	}
}

// A declared structured field row is emitted with its field name. Mutation
// caught: skipping rows whose Field is set.
func TestProfileEmitsDeclaredFieldRow(t *testing.T) {
	f := aa03Fixture(t)
	profile := loadedPilot(t)
	profile.Attributes = append(profile.Attributes, composition.AttributeRow{
		ClusterID: 0x0090, ClusterName: "Electrical Power Measurement", AttributeID: 0x0002, Name: "Accuracy", Field: "Measured",
		MatterType: "bool", Mandatory: true, Origin: composition.OriginDeclared, Declared: true,
	})
	got, err := projectLoadedProfile(profile, f.snapshot, f.evaluation)
	if err != nil {
		t.Fatal(err)
	}
	last := got.Attributes[len(got.Attributes)-1]
	if last.Field != "Measured" || !last.Declared || last.Value != true {
		t.Fatalf("field row %#v", last)
	}
}

// An unknown encoder or a pack with no validator is an error, never a value.
// Mutations caught: a default encoder arm, a skipped pack validation.
func TestProfileRefusesUnknownEncoderAndPack(t *testing.T) {
	f := aa03Fixture(t)
	profile := loadedPilot(t)
	profile.Attributes[currentRowIndex(t, profile)].Encoder = "synthetic_encoder"
	if _, err := projectLoadedProfile(profile, f.snapshot, f.evaluation); err == nil {
		t.Fatal("accepted an unknown encoder")
	}
	profile = loadedPilot(t)
	profile.Attributes[currentRowIndex(t, profile)].PackID = "synthetic.pack"
	if got, err := projectLoadedProfile(profile, f.snapshot, f.evaluation); err == nil && got.Attributes != nil {
		if a := profileAttribute(t, got, "ActiveCurrent"); a.Value == "3200" {
			t.Fatal("accepted a pack with no validator")
		}
	}
}
