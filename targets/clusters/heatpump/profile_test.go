package heatpump

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

var accuracy = []string{"MeasurementType", "Measured", "MinMeasuredValue", "MaxMeasuredValue", "AccuracyRanges", "RangeMin", "RangeMax"}

func withAccuracy(part string, cluster, attribute uint32) []key {
	out := []key{{part, cluster, attribute, ""}}
	for _, f := range accuracy {
		out = append(out, key{part, cluster, attribute, f})
	}
	return out
}

func TestIdentityAndKinds(t *testing.T) {
	if DeviceTypeID != 0x0309 || DeviceTypeRevision != 1 {
		t.Fatal("device type identity")
	}
	if !reflect.DeepEqual(Kinds(), []string{"resource.heat_generator"}) {
		t.Fatalf("kinds %v", Kinds())
	}
}

func TestRequirementsTree(t *testing.T) {
	var nested []partrows.Requirement
	optional := map[string]string{}
	for _, r := range Requirements() {
		if r.Element != "device_type" {
			continue
		}
		if r.Conformance == "M" {
			nested = append(nested, r)
		} else {
			optional[r.Name] = r.Conformance
			if r.Mounted {
				t.Fatalf("optional device type mounted %#v", r)
			}
		}
	}
	if !reflect.DeepEqual(optional, map[string]string{"Thermostat": "O", "Temperature Sensor": "O", "Water Heater": "O"}) {
		t.Fatalf("optional %v", optional)
	}
	if len(nested) != 3 || nested[0].ID != 0x0011 || nested[1].ID != 0x050D || nested[2].ID != 0x0510 {
		t.Fatalf("mandatory nested device types %#v", nested)
	}
	for _, n := range nested {
		if n.Min != 1 || !n.Mounted {
			t.Fatalf("nested %#v", n)
		}
	}
	ps := nested[0].Children
	if len(ps) != 2 || ps[0].ID != 0x002F || ps[0].Children[0].Name != "WIRED" || ps[0].Children[0].Conformance != "M" ||
		ps[1].ID != 0x001D || ps[1].Children[0].Name != "TAGLIST" || ps[1].Children[0].Conformance != "M" {
		t.Fatalf("power source requirements %#v", ps)
	}
	dem := nested[1].Children
	if len(dem) != 1 || dem[0].ID != 0x0098 || len(dem[0].Children) != 1 || dem[0].Children[0].Name != "PA" || dem[0].Children[0].Conformance != "M" {
		t.Fatalf("device energy management requirements %#v", dem)
	}
	es := nested[2].Children
	if len(es) != 3 || es[0].ID != 0x0090 || es[1].ID != 0x0091 || es[2].ID != 0x009C {
		t.Fatalf("electrical sensor requirements %#v", es)
	}
	epm := map[string]string{}
	for _, c := range es[0].Children {
		epm[c.Element+":"+c.Name] = c.Conformance
	}
	if !reflect.DeepEqual(epm, map[string]string{"feature:ALTC": "M", "attribute:Voltage": "M", "attribute:ActiveCurrent": "M"}) {
		t.Fatalf("electrical power measurement requirements %v", epm)
	}
}

func TestPartGeometryMatchesRequirements(t *testing.T) {
	parts := Parts()
	if len(parts) != 3 {
		t.Fatalf("parts %d", len(parts))
	}
	want := map[string]struct {
		device, revision uint32
		servers          []uint32
	}{
		RolePowerSource:      {0x0011, 1, []uint32{0x002F}},
		RoleDeviceEnergyMgmt: {0x050D, 3, []uint32{0x0098}},
		RoleElectricalSensor: {0x0510, 1, []uint32{0x0090, 0x0091, 0x009C}},
	}
	var roles []any
	for _, p := range parts {
		w, ok := want[p.Role]
		if !ok || p.DeviceTypeID != w.device || p.Revision != w.revision || p.Count < 1 || len(p.Children) != 0 {
			t.Fatalf("part %#v", p)
		}
		roles = append(roles, p.Role)
		var got []uint32
		for _, s := range p.Servers {
			got = append(got, s.ClusterID)
		}
		if !reflect.DeepEqual(got, w.servers) {
			t.Fatalf("%s servers %v", p.Role, got)
		}
	}
	if !reflect.DeepEqual(descriptorList("", 0x0003), roles) {
		t.Fatalf("PartsList %v, parts %v", descriptorList("", 0x0003), roles)
	}
	if len(Servers()) != 0 {
		t.Fatalf("root servers %v", rootServers())
	}
}

func TestMandatoryRowsHaveASource(t *testing.T) {
	missing := []key{
		{RolePowerSource, 0x001D, 0x0004, ""},
		{RolePowerSource, 0x002F, 0x0000, ""}, {RolePowerSource, 0x002F, 0x0001, ""}, {RolePowerSource, 0x002F, 0x0002, ""},
		{RolePowerSource, 0x002F, 0x0005, ""}, {RolePowerSource, 0x002F, 0x001F, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0x0000, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0001, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0002, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0x0003, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0004, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0005, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0x0007, ""},
		{RoleElectricalSensor, 0x0090, 0x0000, ""}, {RoleElectricalSensor, 0x0090, 0x0001, ""},
		{RoleElectricalSensor, 0x0090, 0x0004, ""}, {RoleElectricalSensor, 0x0090, 0x0005, ""}, {RoleElectricalSensor, 0x0090, 0x0008, ""},
		{RoleElectricalSensor, 0x0091, 0xFFFC, ""}, {RoleElectricalSensor, 0x009C, 0xFFFC, ""},
	}
	missing = append(missing, withAccuracy(RoleElectricalSensor, 0x0090, 0x0002)...)
	missing = append(missing, withAccuracy(RoleElectricalSensor, 0x0091, 0x0000)...)
	declared := map[key]any{
		{"", 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0309", "revision": 1}},
		{"", 0x001D, 0x0001, ""}: []any{"0x001D"}, {"", 0x001D, 0x0002, ""}: []any{},
		{"", 0x001D, 0x0003, ""}: []any{RolePowerSource, RoleDeviceEnergyMgmt, RoleElectricalSensor},
		{"", 0x001D, 0xFFFC, ""}: 0, {"", 0x001D, 0xFFFD, ""}: 3,
		{RolePowerSource, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0011", "revision": 1}},
		{RolePowerSource, 0x001D, 0x0001, ""}: []any{"0x001D", "0x002F"}, {RolePowerSource, 0x001D, 0x0002, ""}: []any{},
		{RolePowerSource, 0x001D, 0x0003, ""}: []any{}, {RolePowerSource, 0x001D, 0xFFFC, ""}: 1, {RolePowerSource, 0x001D, 0xFFFD, ""}: 3,
		{RolePowerSource, 0x002F, 0xFFFC, ""}: 1, {RolePowerSource, 0x002F, 0xFFFD, ""}: 3,
		{RoleDeviceEnergyMgmt, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x050D", "revision": 3}},
		{RoleDeviceEnergyMgmt, 0x001D, 0x0001, ""}: []any{"0x001D", "0x0098"}, {RoleDeviceEnergyMgmt, 0x001D, 0x0002, ""}: []any{},
		{RoleDeviceEnergyMgmt, 0x001D, 0x0003, ""}: []any{}, {RoleDeviceEnergyMgmt, 0x001D, 0xFFFC, ""}: 0, {RoleDeviceEnergyMgmt, 0x001D, 0xFFFD, ""}: 3,
		{RoleDeviceEnergyMgmt, 0x0098, 0xFFFC, ""}: 1, {RoleDeviceEnergyMgmt, 0x0098, 0xFFFD, ""}: 4,
		{RoleElectricalSensor, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0510", "revision": 1}},
		{RoleElectricalSensor, 0x001D, 0x0001, ""}: []any{"0x001D", "0x0090", "0x0091", "0x009C"},
		{RoleElectricalSensor, 0x001D, 0x0002, ""}: []any{}, {RoleElectricalSensor, 0x001D, 0x0003, ""}: []any{},
		{RoleElectricalSensor, 0x001D, 0xFFFC, ""}: 0, {RoleElectricalSensor, 0x001D, 0xFFFD, ""}: 3,
		{RoleElectricalSensor, 0x0090, 0xFFFC, ""}: 2, {RoleElectricalSensor, 0x0090, 0xFFFD, ""}: 3,
		{RoleElectricalSensor, 0x0091, 0xFFFD, ""}: 2, {RoleElectricalSensor, 0x009C, 0xFFFD, ""}: 1,
	}
	checkMandatory(t, missing, declared)
}
