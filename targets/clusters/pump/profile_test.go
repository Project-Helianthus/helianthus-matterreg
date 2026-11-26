package pump

import (
	"reflect"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

type key struct {
	part      string
	cluster   uint32
	attribute uint32
	field     string
}

func allRows() []composition.AttributeRow {
	rows := Rows()
	for _, p := range Parts() {
		rows = append(rows, p.Attributes...)
	}
	return rows
}

func checkMandatory(t *testing.T, missing []key, declared map[key]any) {
	t.Helper()
	got := map[key]composition.AttributeRow{}
	for _, r := range allRows() {
		if !r.Mandatory {
			continue
		}
		k := key{r.PartKey, r.ClusterID, r.AttributeID, r.Field}
		if _, dup := got[k]; dup {
			t.Fatalf("duplicate row %v", k)
		}
		got[k] = r
		if r.Origin == composition.OriginSemreg {
			t.Fatalf("%s is fed from SemReg though the target maps no mandatory row for this kind", r.ReportName())
		}
		if r.Origin == composition.OriginMissing && (r.Reason != partrows.UnmappedReason || r.Declared != nil) {
			t.Fatalf("%s missing without the target's reason or with a value", r.ReportName())
		}
	}
	if len(got) != len(missing)+len(declared) {
		t.Fatalf("%d rows, want %d", len(got), len(missing)+len(declared))
	}
	for _, k := range missing {
		if got[k].Origin != composition.OriginMissing {
			t.Fatalf("%v is %q", k, got[k].Origin)
		}
	}
	for k, v := range declared {
		if got[k].Origin != composition.OriginDeclared || !reflect.DeepEqual(got[k].Declared, v) {
			t.Fatalf("%v is %q %v, want %v", k, got[k].Origin, got[k].Declared, v)
		}
	}
	if len(MissingRows()) != len(missing) {
		t.Fatalf("%d missing names, want %d", len(MissingRows()), len(missing))
	}
}

func TestNothingIsMappedOrWritableAndNoFoldIsNamed(t *testing.T) {
	for _, r := range allRows() {
		if r.Origin == composition.OriginSemreg || r.Encoder != "" || r.Element != "" || r.Dimension != "" {
			t.Fatalf("%s carries a mapping though the target maps no row for this kind", r.ReportName())
		}
	}
	if len(Folds()) != 0 {
		t.Fatalf("the target names no fold for this kind: %#v", Folds())
	}
}

func TestEveryRowSitsAtItsPart(t *testing.T) {
	for _, r := range Rows() {
		if r.PartKey != "" {
			t.Fatalf("root row %s sits at %q", r.ReportName(), r.PartKey)
		}
	}
	for _, p := range Parts() {
		for _, r := range p.Attributes {
			if r.PartKey != p.Role {
				t.Fatalf("%s row %s sits at %q", p.Role, r.ReportName(), r.PartKey)
			}
		}
	}
}

func rootServers() []uint32 {
	var out []uint32
	for _, s := range Servers() {
		out = append(out, s.ClusterID)
	}
	return out
}

func descriptorList(part string, attribute uint32) any {
	for _, r := range Rows() {
		if r.PartKey == part && r.ClusterID == 0x001D && r.AttributeID == attribute {
			return r.Declared
		}
	}
	return nil
}

func rootTree(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, r := range Requirements() {
		out[r.Element+":"+r.Name] = r.Conformance
	}
	return out
}

func TestIdentityAndKinds(t *testing.T) {
	if DeviceTypeID != 0x0303 || DeviceTypeRevision != 4 {
		t.Fatal("device type identity")
	}
	if !reflect.DeepEqual(Kinds(), []string{"resource.pump"}) {
		t.Fatalf("kinds %v", Kinds())
	}
}

func TestRequirementsTreeAndNoParts(t *testing.T) {
	tree := rootTree(t)
	mandatory := map[string]bool{}
	for k, c := range tree {
		if c == "M" {
			mandatory[k] = true
		}
	}
	wantMandatory := map[string]bool{"server_cluster:Identify": true, "server_cluster:On/Off": true, "server_cluster:Pump Configuration and Control": true}
	if !reflect.DeepEqual(mandatory, wantMandatory) {
		t.Fatalf("mandatory %v", mandatory)
	}
	if len(Requirements()) != 14 {
		t.Fatalf("%d requirements", len(Requirements()))
	}
	for _, r := range Requirements() {
		if r.Element == "device_type" {
			t.Fatalf("the device type nests no device type: %#v", r)
		}
		if r.Mounted != (r.Conformance == "M" || r.Name == "Descriptor") {
			t.Fatalf("mounted flag of %#v", r)
		}
	}
	if len(Parts()) != 0 {
		t.Fatalf("parts %#v", Parts())
	}
	if !reflect.DeepEqual(rootServers(), []uint32{0x0003, 0x0006, 0x0200}) {
		t.Fatalf("root servers %v", rootServers())
	}
}

func TestMandatoryRowsHaveASource(t *testing.T) {
	missing := []key{
		{"", 0x0003, 0x0000, ""}, {"", 0x0003, 0x0001, ""}, {"", 0x0006, 0x0000, ""},
		{"", 0x0200, 0x0000, ""}, {"", 0x0200, 0x0001, ""}, {"", 0x0200, 0x0002, ""}, {"", 0x0200, 0x0011, ""},
		{"", 0x0200, 0x0012, ""}, {"", 0x0200, 0x0013, ""}, {"", 0x0200, 0x0020, ""}, {"", 0x0200, 0xFFFC, ""},
	}
	declared := map[key]any{
		{"", 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0303", "revision": 4}},
		{"", 0x001D, 0x0001, ""}: []any{"0x001D", "0x0003", "0x0006", "0x0200"}, {"", 0x001D, 0x0002, ""}: []any{},
		{"", 0x001D, 0x0003, ""}: []any{}, {"", 0x001D, 0xFFFC, ""}: 0, {"", 0x001D, 0xFFFD, ""}: 3,
		{"", 0x0003, 0xFFFC, ""}: 0, {"", 0x0003, 0xFFFD, ""}: 6,
		{"", 0x0006, 0xFFFC, ""}: 0, {"", 0x0006, 0xFFFD, ""}: 6,
		{"", 0x0200, 0xFFFD, ""}: 5,
	}
	checkMandatory(t, missing, declared)
}
