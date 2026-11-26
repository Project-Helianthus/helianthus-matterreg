package electricalsensor

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/powertopology"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

type key struct {
	cluster   uint32
	attribute uint32
	field     string
}

func TestKindsAndDeviceType(t *testing.T) {
	if !reflect.DeepEqual(Kinds(), []string{"resource.meter", "resource.grid_connection", "resource.electrical_circuit"}) {
		t.Fatalf("kinds %v", Kinds())
	}
	if DeviceTypeID != 0x0510 || DeviceTypeRevision != 1 {
		t.Fatal("device type identity")
	}
	for _, kind := range Kinds() {
		if kind == pilot.ResourceKind {
			t.Fatalf("kind %s duplicates the pilot kind", kind)
		}
	}
}

func TestRequirementsTree(t *testing.T) {
	want := []Requirement{
		{0x001D, "Descriptor", "M", true, true, ""},
		{0x0090, "Electrical Power Measurement", "O.a+", true, false, "satisfies the at-least-one choice of Electrical Power Measurement and Electrical Energy Measurement"},
		{0x0091, "Electrical Energy Measurement", "O.a+", false, false, ""},
		{0x009C, "Power Topology", "M", true, false, ""},
	}
	if !reflect.DeepEqual(Requirements(), want) {
		t.Fatalf("requirements %#v", Requirements())
	}
	mounted := 0
	for _, r := range Requirements() {
		if r.Mounted && r.Conformance == "O.a+" {
			mounted++
		}
	}
	if mounted < 1 {
		t.Fatal("the at-least-one choice has no mounted cluster")
	}
}

func TestMandatoryRowsHaveASource(t *testing.T) {
	missing := []key{
		{0x009C, 0xFFFC, ""}, {0x0090, 0xFFFC, ""},
		{0x0090, 0x0000, ""}, {0x0090, 0x0001, ""}, {0x0090, 0x0002, ""},
		{0x0090, 0x0002, "MeasurementType"}, {0x0090, 0x0002, "Measured"},
		{0x0090, 0x0002, "MinMeasuredValue"}, {0x0090, 0x0002, "MaxMeasuredValue"},
		{0x0090, 0x0002, "AccuracyRanges"}, {0x0090, 0x0002, "RangeMin"}, {0x0090, 0x0002, "RangeMax"},
		{0x0090, 0x0008, ""},
	}
	declared := []key{
		{0x001D, 0x0000, ""}, {0x001D, 0x0001, ""}, {0x001D, 0x0002, ""}, {0x001D, 0x0003, ""},
		{0x001D, 0xFFFC, ""}, {0x001D, 0xFFFD, ""}, {0x009C, 0xFFFD, ""}, {0x0090, 0xFFFD, ""},
	}
	got := map[key]pilot.Origin{}
	for _, row := range Rows() {
		if !row.Mandatory {
			t.Fatalf("optional row %s listed", row.ReportName())
		}
		k := key{row.ClusterID, row.AttributeID, row.Field}
		if _, dup := got[k]; dup {
			t.Fatalf("duplicate row %v", k)
		}
		got[k] = row.Origin
		if row.Origin == pilot.OriginSemreg {
			t.Fatalf("%s is fed from SemReg though the target maps no row for these kinds", row.ReportName())
		}
		if row.Origin == pilot.OriginDeclared && row.Declared == nil {
			t.Fatalf("%s declared without a value", row.ReportName())
		}
		if row.Origin == pilot.OriginMissing && row.Detail == "" {
			t.Fatalf("%s missing without a named reason", row.ReportName())
		}
	}
	if len(got) != len(missing)+len(declared) {
		t.Fatalf("%d rows, want %d", len(got), len(missing)+len(declared))
	}
	for _, k := range missing {
		if got[k] != pilot.OriginMissing {
			t.Fatalf("%v is %q", k, got[k])
		}
	}
	for _, k := range declared {
		if got[k] != pilot.OriginDeclared {
			t.Fatalf("%v is %q", k, got[k])
		}
	}
	names := MissingRows()
	sort.Strings(names)
	if len(names) != len(missing) || names[0] == "" {
		t.Fatalf("missing names %v", names)
	}
}

func TestEveryMandatoryAttributeOfMountedClustersHasARow(t *testing.T) {
	rows := map[key]bool{}
	for _, row := range Rows() {
		rows[key{row.ClusterID, row.AttributeID, ""}] = true
	}
	tables := map[uint32]clusters.Table{0x0090: electricalpowermeasurement.Table(), 0x009C: powertopology.Table()}
	for _, r := range Requirements() {
		if !r.Mounted || r.SDKSupplied {
			continue
		}
		table, ok := tables[r.ClusterID]
		if !ok {
			t.Fatalf("no table for %s", r.Name)
		}
		for _, a := range table.Attributes {
			if a.Mandatory && !rows[key{table.ClusterID, a.ID, ""}] {
				t.Fatalf("%s.%s has no row", table.Name, a.Name)
			}
		}
	}
}

func TestNoTransformedRowForTheseKinds(t *testing.T) {
	for _, row := range Rows() {
		if row.AttributeID == 0x0005 {
			t.Fatalf("ActiveCurrent row %s has no mapping for these kinds", row.ReportName())
		}
		if row.Origin == pilot.OriginMissing && row.AttributeID < 0xFFF0 && row.Reason != UnmappedReason {
			t.Fatalf("%s reason %q", row.ReportName(), row.Reason)
		}
	}
}

func TestFoldsAreDeclaredLosses(t *testing.T) {
	folds := Folds()
	if len(folds) != 1 || folds[0].Name != "phase" || folds[0].Kind != "identity" || !folds[0].Declared || len(folds[0].SourceItems) != 1 {
		t.Fatalf("folds %#v", folds)
	}
}
