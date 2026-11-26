package batterystorage

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

func sensorMissing(role string) []key {
	m := []key{
		{role, 0x001D, 0x0004, ""},
		{role, 0x0090, 0x0000, ""}, {role, 0x0090, 0x0001, ""}, {role, 0x0090, 0x0002, ""},
		{role, 0x0090, 0x0004, ""}, {role, 0x0090, 0x0005, ""}, {role, 0x0090, 0x0008, ""},
		{role, 0x0091, 0xFFFC, ""}, {role, 0x0091, 0x0000, ""}, {role, 0x009C, 0xFFFC, ""},
	}
	for _, f := range accuracy {
		m = append(m, key{role, 0x0090, 0x0002, f}, key{role, 0x0091, 0x0000, f})
	}
	return m
}

func sourceMissing(role string, ids ...uint32) []key {
	m := []key{{role, 0x001D, 0x0004, ""}}
	for _, id := range ids {
		m = append(m, key{role, 0x002F, id, ""})
	}
	return m
}

func wantMissing() []key {
	m := sensorMissing(RoleElectricalSensorAC)
	m = append(m, sensorMissing(RoleElectricalSensorDC)...)
	m = append(m, sourceMissing(RolePowerSourceWired, 0x0000, 0x0001, 0x0002, 0x0005, 0x001F)...)
	m = append(m, sourceMissing(RolePowerSourceBattery,
		0x0000, 0x0001, 0x0002, 0x001F, 0x000E, 0x000F, 0x0010,
		0x000B, 0x000C, 0x000D, 0x0012, 0x0018, 0x001B, 0x001D, 0x001E, 0x001A, 0x001C)...)
	return m
}

func descriptor(role string, device string, servers []any) map[key]any {
	return map[key]any{
		{role, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": device, "revision": 1}},
		{role, 0x001D, 0x0001, ""}: servers, {role, 0x001D, 0x0002, ""}: []any{}, {role, 0x001D, 0x0003, ""}: []any{},
		{role, 0x001D, 0xFFFC, ""}: 1, {role, 0x001D, 0xFFFD, ""}: 3,
	}
}

func wantDeclared() map[key]any {
	out := map[key]any{
		{"", 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0018", "revision": 2}},
		{"", 0x001D, 0x0001, ""}: []any{"0x001D"}, {"", 0x001D, 0x0002, ""}: []any{},
		{"", 0x001D, 0x0003, ""}: []any{RoleElectricalSensorAC, RoleElectricalSensorDC, RolePowerSourceWired, RolePowerSourceBattery},
		{"", 0x001D, 0xFFFC, ""}: 0, {"", 0x001D, 0xFFFD, ""}: 3,
		{RoleElectricalSensorAC, 0x0090, 0xFFFC, ""}: 2, {RoleElectricalSensorDC, 0x0090, 0xFFFC, ""}: 1,
		{RolePowerSourceWired, 0x002F, 0xFFFC, ""}: 1, {RolePowerSourceBattery, 0x002F, 0xFFFC, ""}: 6,
		{RolePowerSourceWired, 0x002F, 0xFFFD, ""}: 3, {RolePowerSourceBattery, 0x002F, 0xFFFD, ""}: 3,
	}
	for _, role := range []string{RoleElectricalSensorAC, RoleElectricalSensorDC} {
		for k, v := range descriptor(role, "0x0510", []any{"0x001D", "0x0090", "0x0091", "0x009C"}) {
			out[k] = v
		}
		out[key{role, 0x0090, 0xFFFD, ""}] = 3
		out[key{role, 0x0091, 0xFFFD, ""}] = 2
		out[key{role, 0x009C, 0xFFFD, ""}] = 1
	}
	for _, role := range []string{RolePowerSourceWired, RolePowerSourceBattery} {
		for k, v := range descriptor(role, "0x0011", []any{"0x001D", "0x002F"}) {
			out[k] = v
		}
	}
	return out
}

func allRows() []composition.AttributeRow {
	rows := Rows()
	for _, p := range Parts() {
		rows = append(rows, p.Attributes...)
	}
	return rows
}

func TestIdentityAndKinds(t *testing.T) {
	if DeviceTypeID != 0x0018 || DeviceTypeRevision != 2 {
		t.Fatal("device type identity")
	}
	if !reflect.DeepEqual(Kinds(), []string{"resource.battery"}) {
		t.Fatalf("kinds %v", Kinds())
	}
}

func TestRequirementsTree(t *testing.T) {
	counts := map[uint32]int{}
	features := map[string][]string{}
	for _, r := range Requirements() {
		if r.Element != "device_type" || r.Conformance != "M" {
			continue
		}
		if r.Min != 2 {
			t.Fatalf("%s multiplicity %d", r.Name, r.Min)
		}
		counts[r.ID]++
		for _, c := range r.Children {
			for _, f := range c.Children {
				if f.Element == "feature" && f.Name != "TAGLIST" {
					features[r.Name] = append(features[r.Name], f.Name)
				}
			}
		}
	}
	if !reflect.DeepEqual(counts, map[uint32]int{0x0510: 2, 0x0011: 2}) {
		t.Fatalf("instances %v", counts)
	}
	if !reflect.DeepEqual(features["Electrical Sensor"], []string{"ALTC", "EXPE", "DIRC", "EXPE"}) ||
		!reflect.DeepEqual(features["Power Source"], []string{"WIRED", "BAT"}) {
		t.Fatalf("features %v", features)
	}
	for _, r := range Requirements() {
		if (r.Name == "Solar Power" || r.Name == "Temperature Sensor") && (r.Mounted || r.Conformance != "O") {
			t.Fatalf("%s %#v", r.Name, r)
		}
	}
}

func TestPartGeometryMatchesRequirements(t *testing.T) {
	want := []struct {
		role    string
		device  uint32
		servers []uint32
	}{
		{RoleElectricalSensorAC, 0x0510, []uint32{0x0090, 0x0091, 0x009C}},
		{RoleElectricalSensorDC, 0x0510, []uint32{0x0090, 0x0091, 0x009C}},
		{RolePowerSourceWired, 0x0011, []uint32{0x002F}},
		{RolePowerSourceBattery, 0x0011, []uint32{0x002F}},
	}
	parts := Parts()
	if len(parts) != len(want) {
		t.Fatalf("%d parts", len(parts))
	}
	for i, w := range want {
		p := parts[i]
		if p.Role != w.role || p.DeviceTypeID != w.device || p.Revision != 1 || p.Count < 1 || len(p.Children) != 0 {
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
		if !r.Mandatory || r.Origin == composition.OriginSemreg || r.Encoder != "" || r.Element != "" {
			t.Fatalf("%s mandatory %v origin %q: the target maps no row for this kind", r.ReportName(), r.Mandatory, r.Origin)
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

func TestFoldsNameNothingTheTargetDoesNotName(t *testing.T) {
	if len(Folds()) != 0 {
		t.Fatalf("folds %#v", Folds())
	}
}
