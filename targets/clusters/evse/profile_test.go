package evse

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
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

func withAccuracy(part string, cluster, attribute uint32) []key {
	out := []key{{part, cluster, attribute, ""}}
	for _, f := range accuracy {
		out = append(out, key{part, cluster, attribute, f})
	}
	return out
}

func wantMissing() []key {
	m := []key{
		{"", 0x0099, 0x0000, ""}, {"", 0x0099, 0x0001, ""}, {"", 0x0099, 0x0002, ""}, {"", 0x0099, 0x0003, ""},
		{"", 0x0099, 0x0005, ""}, {"", 0x0099, 0x0006, ""}, {"", 0x0099, 0x0007, ""},
		{"", 0x0099, 0x0023, ""}, {"", 0x0099, 0x0024, ""}, {"", 0x0099, 0x0025, ""}, {"", 0x0099, 0x0026, ""},
		{"", 0x0099, 0x0040, ""}, {"", 0x0099, 0x0041, ""}, {"", 0x0099, 0x0042, ""},
		{"", 0x009D, 0x0000, ""}, {"", 0x009D, 0x0001, ""},
		{RolePowerSource, 0x002F, 0x0000, ""}, {RolePowerSource, 0x002F, 0x0001, ""}, {RolePowerSource, 0x002F, 0x0002, ""},
		{RolePowerSource, 0x002F, 0x001F, ""}, {RolePowerSource, 0x002F, 0xFFFC, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0x0000, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0001, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0002, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0x0003, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0004, ""}, {RoleDeviceEnergyMgmt, 0x0098, 0x0006, ""},
		{RoleDeviceEnergyMgmt, 0x0098, 0xFFFC, ""},
		{RoleElectricalSensor, 0x0090, 0xFFFC, ""}, {RoleElectricalSensor, 0x0090, 0x0000, ""}, {RoleElectricalSensor, 0x0090, 0x0001, ""},
		{RoleElectricalSensor, 0x0090, 0x0008, ""},
		{RoleElectricalSensor, 0x0091, 0xFFFC, ""},
		{RoleElectricalSensor, 0x009C, 0xFFFC, ""},
	}
	m = append(m, withAccuracy(RoleElectricalSensor, 0x0090, 0x0002)...)
	m = append(m, withAccuracy(RoleElectricalSensor, 0x0091, 0x0000)...)
	return m
}

func wantDeclared() map[key]any {
	return map[key]any{
		{"", 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x050C", "revision": 2}},
		{"", 0x001D, 0x0001, ""}: []any{"0x001D", "0x0099", "0x009D"}, {"", 0x001D, 0x0002, ""}: []any{},
		{"", 0x001D, 0x0003, ""}: []any{RolePowerSource, RoleDeviceEnergyMgmt, RoleElectricalSensor},
		{"", 0x001D, 0xFFFC, ""}: 0, {"", 0x001D, 0xFFFD, ""}: 3,
		{"", 0x0099, 0xFFFC, ""}: 1, {"", 0x0099, 0xFFFD, ""}: 4,
		{"", 0x009D, 0xFFFC, ""}: 0, {"", 0x009D, 0xFFFD, ""}: 2,
		{RolePowerSource, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x0011", "revision": 1}},
		{RolePowerSource, 0x001D, 0x0001, ""}: []any{"0x001D", "0x002F"}, {RolePowerSource, 0x001D, 0x0002, ""}: []any{},
		{RolePowerSource, 0x001D, 0x0003, ""}: []any{}, {RolePowerSource, 0x001D, 0xFFFC, ""}: 0, {RolePowerSource, 0x001D, 0xFFFD, ""}: 3,
		{RolePowerSource, 0x002F, 0xFFFD, ""}:      3,
		{RoleDeviceEnergyMgmt, 0x001D, 0x0000, ""}: []any{map[string]any{"device_type": "0x050D", "revision": 3}},
		{RoleDeviceEnergyMgmt, 0x001D, 0x0001, ""}: []any{"0x001D", "0x0098"}, {RoleDeviceEnergyMgmt, 0x001D, 0x0002, ""}: []any{},
		{RoleDeviceEnergyMgmt, 0x001D, 0x0003, ""}: []any{}, {RoleDeviceEnergyMgmt, 0x001D, 0xFFFC, ""}: 0, {RoleDeviceEnergyMgmt, 0x001D, 0xFFFD, ""}: 3,
		{RoleDeviceEnergyMgmt, 0x0098, 0xFFFD, ""}: 4,
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
	if DeviceTypeID != 0x050C || DeviceTypeRevision != 2 {
		t.Fatal("device type identity")
	}
	if !reflect.DeepEqual(Kinds(), []string{"resource.charging_station", "resource.evse"}) {
		t.Fatalf("kinds %v", Kinds())
	}
}

func TestRequirementsTree(t *testing.T) {
	tree := Requirements()
	var nested []partrows.Requirement
	rootClusters := map[uint32]string{}
	for _, r := range tree {
		if r.Element == "device_type" && r.Conformance == "M" {
			nested = append(nested, r)
		}
		if r.Element == "server_cluster" {
			rootClusters[r.ID] = r.Conformance
		}
	}
	if len(nested) != 3 || nested[0].ID != 0x0011 || nested[1].ID != 0x050D || nested[2].ID != 0x0510 {
		t.Fatalf("mandatory nested device types %#v", nested)
	}
	for _, n := range nested {
		if n.Min != 1 || !n.Mounted {
			t.Fatalf("nested %#v", n)
		}
	}
	if !reflect.DeepEqual(rootClusters, map[uint32]string{0x001D: "", 0x0003: "O", 0x0099: "M", 0x009D: "M", 0x0402: "O"}) {
		t.Fatalf("root clusters %v", rootClusters)
	}
	dem := nested[1].Children[0]
	if dem.ID != 0x0098 || len(dem.Children) != 2 || dem.Children[0].Name != "PFR" || dem.Children[0].Conformance != "M" ||
		dem.Children[1].Name != "PA" || dem.Children[1].Conformance != "desc" || dem.Children[1].Mounted {
		t.Fatalf("device energy management requirement %#v", dem)
	}
	sensor := map[uint32]string{}
	for _, c := range nested[2].Children {
		sensor[c.ID] = c.Conformance
	}
	if !reflect.DeepEqual(sensor, map[uint32]string{0x0090: "M", 0x0091: "M", 0x009C: "M"}) {
		t.Fatalf("electrical sensor requirements %v", sensor)
	}
	for _, r := range tree {
		if r.Name == "Temperature Measurement" && (r.Mounted || r.Conformance != "O") {
			t.Fatalf("temperature measurement %#v", r)
		}
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
	var roles []string
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
	var rootParts any
	for _, r := range Rows() {
		if r.ClusterID == 0x001D && r.AttributeID == 0x0003 {
			rootParts = r.Declared
		}
	}
	if !reflect.DeepEqual(rootParts, []any{roles[0], roles[1], roles[2]}) {
		t.Fatalf("PartsList %v, parts %v", rootParts, roles)
	}
	var rootServers []uint32
	for _, s := range Servers() {
		rootServers = append(rootServers, s.ClusterID)
	}
	if !reflect.DeepEqual(rootServers, []uint32{0x0099, 0x009D}) {
		t.Fatalf("root servers %v", rootServers)
	}
}

func TestMandatoryRowsHaveASource(t *testing.T) {
	missing := wantMissing()
	declared := wantDeclared()
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
			t.Fatalf("%s is fed from SemReg though the target maps no mandatory row for these kinds", r.ReportName())
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

func TestOnlyActiveCurrentIsMappedAndNothingIsWritable(t *testing.T) {
	mapped := 0
	for _, r := range allRows() {
		if r.Origin != composition.OriginSemreg {
			if r.Encoder != "" || r.Element != "" {
				t.Fatalf("%s carries a mapping though it is not fed from SemReg", r.ReportName())
			}
			continue
		}
		mapped++
		if r.PartKey != RoleElectricalSensor || r.ClusterID != 0x0090 || r.AttributeID != 0x0005 || r.Mandatory ||
			r.Element != "evse.ac.current" || r.Dimension != "evse.dimension.phase" || r.Encoder != "ampere_to_milliampere" ||
			r.PackID != "helianthus.pack.evse" || r.PackVersion != "1.0.0" || r.MatterType != "amperage-mA" || !r.Nullable {
			t.Fatalf("mapped row %#v", r)
		}
	}
	if mapped != 1 {
		t.Fatalf("%d mapped rows", mapped)
	}
}

func TestActiveCurrentKeepsItsExactOrReportedEncoding(t *testing.T) {
	table := electricalpowermeasurement.Table()
	if len(table.Mappings) != 1 || table.Mappings[0].Encoder != "ampere_to_milliampere" || table.Mappings[0].Rounding != "exact" || !table.Mappings[0].RequireIntegral {
		t.Fatalf("mapping %#v", table.Mappings)
	}
	exact := []struct {
		coefficient string
		exponent    int
		want        string
	}{{"32", 0, "32000"}, {"-16", 0, "-16000"}, {"0", 0, "0"}, {"125", -2, "1250"}, {"5", 1, "50000"}, {"1", -3, "1"}}
	for _, c := range exact {
		got, err := electricalpowermeasurement.AmpereToMilliampere(c.coefficient, c.exponent)
		if err != nil || got != c.want {
			t.Fatalf("%sE%d: %q %v, want %q", c.coefficient, c.exponent, got, err, c.want)
		}
	}
	for _, c := range []struct {
		coefficient string
		exponent    int
	}{{"1", -4}, {"1", -5}, {"12345", -7}} {
		got, err := electricalpowermeasurement.AmpereToMilliampere(c.coefficient, c.exponent)
		if !errors.Is(err, electricalpowermeasurement.ErrNotIntegral) || got != "" {
			t.Fatalf("%sE%d must be reported and not rounded: %q %v", c.coefficient, c.exponent, got, err)
		}
	}
}

func foldSame(a, b composition.Fold) bool {
	return a.Name == b.Name && a.Kind == b.Kind && a.Declared == b.Declared && reflect.DeepEqual(a.SourceItems, b.SourceItems)
}

func TestFoldsAreDeclaredLosses(t *testing.T) {
	want := map[string]composition.Fold{
		"connector": {Name: "connector", Kind: "identity", SourceItems: []string{"evse.dimension.connector"}, Declared: true},
		"phase":     {Name: "phase", Kind: "identity", SourceItems: []string{"evse.dimension.phase"}, Declared: true},
	}
	folds := Folds()
	if len(folds) != len(want) {
		t.Fatalf("folds %#v", folds)
	}
	for _, f := range folds {
		w, ok := want[f.Name]
		if !ok || !foldSame(f, w) || f.Description == "" {
			t.Fatalf("fold %#v", f)
		}
	}
}

func TestConnectorFoldIdentityFieldsAreEachChecked(t *testing.T) {
	var connector composition.Fold
	for _, f := range Folds() {
		if f.Name == "connector" {
			connector = f
		}
	}
	mutations := map[string]func(*composition.Fold){
		"Name":        func(f *composition.Fold) { f.Name = "phase" },
		"Kind":        func(f *composition.Fold) { f.Kind = "value" },
		"SourceItems": func(f *composition.Fold) { f.SourceItems = []string{"evse.dimension.phase"} },
		"Declared":    func(f *composition.Fold) { f.Declared = false },
	}
	for field, mutate := range mutations {
		changed := connector
		changed.SourceItems = append([]string(nil), connector.SourceItems...)
		mutate(&changed)
		if foldSame(connector, changed) {
			t.Fatalf("a change of %s is not caught", field)
		}
	}
}

func TestEveryMandatoryAttributeOfMountedClustersHasARow(t *testing.T) {
	rows := map[key]bool{}
	for _, r := range allRows() {
		rows[key{r.PartKey, r.ClusterID, r.AttributeID, ""}] = true
	}
	mandatory := []struct {
		part    string
		cluster uint32
		ids     []uint32
	}{
		{"", 0x0099, []uint32{0x0000, 0x0001, 0x0002, 0x0003, 0x0005, 0x0006, 0x0007, 0x0023, 0x0024, 0x0025, 0x0026, 0x0040, 0x0041, 0x0042, 0xFFFC, 0xFFFD}},
		{"", 0x009D, []uint32{0x0000, 0x0001, 0xFFFC, 0xFFFD}},
		{RolePowerSource, 0x002F, []uint32{0x0000, 0x0001, 0x0002, 0x001F, 0xFFFC, 0xFFFD}},
		{RoleDeviceEnergyMgmt, 0x0098, []uint32{0x0000, 0x0001, 0x0002, 0x0003, 0x0004, 0x0006, 0xFFFC, 0xFFFD}},
		{RoleElectricalSensor, 0x0090, []uint32{0x0000, 0x0001, 0x0002, 0x0008, 0xFFFC, 0xFFFD}},
		{RoleElectricalSensor, 0x0091, []uint32{0x0000, 0xFFFC, 0xFFFD}},
		{RoleElectricalSensor, 0x009C, []uint32{0xFFFC, 0xFFFD}},
	}
	for _, c := range mandatory {
		for _, id := range c.ids {
			if !rows[key{c.part, c.cluster, id, ""}] {
				t.Fatalf("%q cluster %#x attribute %#x has no row", c.part, c.cluster, id)
			}
		}
	}
}
