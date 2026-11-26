package solarpower

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

var accuracy = []string{"MeasurementType", "Measured", "MinMeasuredValue", "MaxMeasuredValue", "AccuracyRanges", "RangeMin", "RangeMax"}

func withAccuracy(part string, cluster uint32) []key {
	out := []key{}
	for _, f := range accuracy {
		out = append(out, key{part, cluster, 0x0002, f})
	}
	return out
}

func wantMissing() []key {
	m := []key{
		{RolePowerSource, 0x001D, 0x0004, ""},
		{RolePowerSource, 0x002F, 0x0000, ""}, {RolePowerSource, 0x002F, 0x0001, ""}, {RolePowerSource, 0x002F, 0x0002, ""},
		{RolePowerSource, 0x002F, 0x0005, ""}, {RolePowerSource, 0x002F, 0x001F, ""},
		{RoleElectricalSensor, 0x0090, 0xFFFC, ""}, {RoleElectricalSensor, 0x0090, 0x0000, ""}, {RoleElectricalSensor, 0x0090, 0x0001, ""},
		{RoleElectricalSensor, 0x0090, 0x0002, ""}, {RoleElectricalSensor, 0x0090, 0x0004, ""}, {RoleElectricalSensor, 0x0090, 0x0005, ""},
		{RoleElectricalSensor, 0x0090, 0x0008, ""},
		{RoleElectricalSensor, 0x0091, 0xFFFC, ""}, {RoleElectricalSensor, 0x0091, 0x0000, ""},
		{RoleElectricalSensor, 0x009C, 0xFFFC, ""},
	}
	for _, f := range accuracy {
		m = append(m, key{RoleElectricalSensor, 0x0090, 0x0002, f}, key{RoleElectricalSensor, 0x0091, 0x0000, f})
	}
	return m
}

func wantDeclared() map[key]any {
	return map[key]any{
		{"", 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0017", "revision": 1}},
		{"", 0x001D, 0x0001, ""}: []any{"0x001D"}, {"", 0x001D, 0x0002, ""}: []any{},
		{"", 0x001D, 0x0003, ""}: []any{RolePowerSource, RoleElectricalSensor},
		{"", 0x001D, 0xFFFC, ""}: 0, {"", 0x001D, 0xFFFD, ""}: 3,
		{RolePowerSource, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0011", "revision": 1}},
		{RolePowerSource, 0x001D, 0x0001, ""}: []any{"0x001D", "0x002F"}, {RolePowerSource, 0x001D, 0x0002, ""}: []any{},
		{RolePowerSource, 0x001D, 0x0003, ""}: []any{}, {RolePowerSource, 0x001D, 0xFFFC, ""}: 1, {RolePowerSource, 0x001D, 0xFFFD, ""}: 3,
		{RolePowerSource, 0x002F, 0xFFFC, ""}: 1, {RolePowerSource, 0x002F, 0xFFFD, ""}: 3,
		{RoleElectricalSensor, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0510", "revision": 1}},
		{RoleElectricalSensor, 0x001D, 0x0001, ""}: []any{"0x001D", "0x0090", "0x0091", "0x009C"},
		{RoleElectricalSensor, 0x001D, 0x0002, ""}: []any{}, {RoleElectricalSensor, 0x001D, 0x0003, ""}: []any{},
		{RoleElectricalSensor, 0x001D, 0xFFFC, ""}: 0, {RoleElectricalSensor, 0x001D, 0xFFFD, ""}: 3,
		{RoleElectricalSensor, 0x0090, 0xFFFD, ""}: 3, {RoleElectricalSensor, 0x0091, 0xFFFD, ""}: 2, {RoleElectricalSensor, 0x009C, 0xFFFD, ""}: 1,
	}
}

func allRows() []composition.AttributeRow {
	rows := Rows()
	for _, p := range Parts() {
		rows = append(rows, p.Attributes...)
	}
	return rows
}

func TestIdentityAndKinds(t *testing.T) {
	if DeviceTypeID != 0x0017 || DeviceTypeRevision != 1 {
		t.Fatal("device type identity")
	}
	if !reflect.DeepEqual(Kinds(), []string{"resource.inverter", "resource.pv_system"}) {
		t.Fatalf("kinds %v", Kinds())
	}
}

func TestRequirementsTree(t *testing.T) {
	tree := Requirements()
	var nested []partrows.Requirement
	for _, r := range tree {
		if r.Element == "device_type" && r.Conformance == "M" {
			nested = append(nested, r)
		}
	}
	if len(nested) != 2 || nested[0].ID != 0x0011 || nested[1].ID != 0x0510 || nested[0].Min != 1 || nested[1].Min != 1 {
		t.Fatalf("mandatory nested device types %#v", nested)
	}
	powerSource := nested[0].Children[0]
	if powerSource.ID != 0x002F || powerSource.Children[0].Name != "WIRED" || powerSource.Children[0].Conformance != "M" {
		t.Fatalf("power source requirement %#v", powerSource)
	}
	var energy, power partrows.Requirement
	for _, c := range nested[1].Children {
		switch c.ID {
		case 0x0091:
			energy = c
		case 0x0090:
			power = c
		}
	}
	if energy.Children[0].Name != "EXPE" || len(power.Children) != 2 || power.Children[0].Name != "Voltage" || power.Children[1].Name != "ActiveCurrent" {
		t.Fatalf("electrical sensor requirements %#v %#v", power, energy)
	}
	for _, r := range tree {
		if r.Name == "Device Energy Management" && (r.Mounted || r.Conformance != "desc") {
			t.Fatalf("device energy management %#v", r)
		}
		if r.Name == "Temperature Sensor" && (r.Mounted || r.Conformance != "O") {
			t.Fatalf("temperature sensor %#v", r)
		}
	}
}

func TestPartGeometryMatchesRequirements(t *testing.T) {
	parts := Parts()
	if len(parts) != 2 {
		t.Fatalf("parts %d", len(parts))
	}
	want := map[string]struct {
		device, revision uint32
		servers          []uint32
	}{
		RolePowerSource:      {0x0011, 1, []uint32{0x002F}},
		RoleElectricalSensor: {0x0510, 1, []uint32{0x0090, 0x0091, 0x009C}},
	}
	for _, p := range parts {
		w, ok := want[p.Role]
		if !ok || p.DeviceTypeID != w.device || p.Revision != w.revision || p.Count < 1 || len(p.Children) != 0 {
			t.Fatalf("part %#v", p)
		}
		var got []uint32
		for _, s := range p.Servers {
			got = append(got, s.ClusterID)
		}
		if !reflect.DeepEqual(got, w.servers) {
			t.Fatalf("%s servers %v", p.Role, got)
		}
		for _, r := range p.Attributes {
			if r.PartKey != p.Role {
				t.Fatalf("%s row %s sits at %q", p.Role, r.ReportName(), r.PartKey)
			}
		}
	}
	for _, r := range Rows() {
		if r.PartKey != "" {
			t.Fatalf("root row %s sits at %q", r.ReportName(), r.PartKey)
		}
	}
}

func TestMandatoryRowsHaveASource(t *testing.T) {
	missing := wantMissing()
	declared := wantDeclared()
	got := map[key]composition.AttributeRow{}
	for _, r := range allRows() {
		k := key{r.PartKey, r.ClusterID, r.AttributeID, r.Field}
		if _, dup := got[k]; dup {
			t.Fatalf("duplicate row %v", k)
		}
		got[k] = r
		if !r.Mandatory {
			t.Fatalf("optional row %s listed", r.ReportName())
		}
		if r.Origin == composition.OriginSemreg {
			t.Fatalf("%s is fed from SemReg though the target maps no row for these kinds", r.ReportName())
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

func TestNoRowIsMappedOrWritable(t *testing.T) {
	for _, r := range allRows() {
		if r.AttributeID == 0x0005 && r.ClusterID == 0x0090 && r.Origin != composition.OriginMissing {
			t.Fatalf("ActiveCurrent %s has no mapping for these kinds", r.PartKey)
		}
		if r.Encoder != "" || r.Element != "" {
			t.Fatalf("%s carries a mapping", r.ReportName())
		}
	}
}

func TestFoldsAreDeclaredLosses(t *testing.T) {
	want := map[string]string{"dc_input": "pv.dimension.input", "phase": "pv.dimension.phase"}
	folds := Folds()
	if len(folds) != len(want) {
		t.Fatalf("folds %#v", folds)
	}
	for _, f := range folds {
		if want[f.Name] == "" || f.Kind != "identity" || !f.Declared || !reflect.DeepEqual(f.SourceItems, []string{want[f.Name]}) {
			t.Fatalf("fold %#v", f)
		}
	}
}

func TestEveryMandatoryAttributeOfMountedClustersHasARow(t *testing.T) {
	rows := map[key]bool{}
	for _, r := range allRows() {
		rows[key{r.PartKey, r.ClusterID, r.AttributeID, ""}] = true
	}
	mandatory := map[string][]uint32{
		RolePowerSource:      {0x002F, 0x0000, 0x0001, 0x0002, 0x0005, 0x001F, 0xFFFC, 0xFFFD},
		RoleElectricalSensor: {0x0090, 0x0000, 0x0001, 0x0002, 0x0004, 0x0005, 0x0008, 0xFFFC, 0xFFFD},
	}
	for part, ids := range mandatory {
		for _, id := range ids[1:] {
			if !rows[key{part, ids[0], id, ""}] {
				t.Fatalf("%s cluster %#x attribute %#x has no row", part, ids[0], id)
			}
		}
	}
	for _, c := range []struct {
		cluster uint32
		id      []uint32
	}{{0x0091, []uint32{0x0000, 0xFFFC, 0xFFFD}}, {0x009C, []uint32{0xFFFC, 0xFFFD}}} {
		for _, id := range c.id {
			if !rows[key{RoleElectricalSensor, c.cluster, id, ""}] {
				t.Fatalf("cluster %#x attribute %#x has no row", c.cluster, id)
			}
		}
	}
}
