package node_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	descriptordef "thingfabric.dev/go-fabric/cluster/spec/descriptor"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/fabrictest"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
	"github.com/Project-Helianthus/helianthus-matterreg/publication"
)

var (
	bounded    = nodetest.Bounded
	measured   = nodetest.Measured
	zone       = nodetest.Zone
	assembly   = nodetest.Assembly
	newFixture = nodetest.NewFixture
)

func TestCommissionReadsMandatoryAttributesOverCASE(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	bridge := f.Start(ctx)
	f.Apply(ctx, zone("zone-a", 2100))
	dev, err := f.Launcher.Last()
	if err != nil {
		t.Fatal(err)
	}
	if w := dev.Node().Window(); !w.Open {
		t.Fatalf("no commissioning window is open on a node without a fabric: %+v", w)
	}
	if _, err := os.Stat(filepath.Join(f.Dir, node.StoreFile)); err != nil {
		t.Fatalf("store file: %v", err)
	}
	entries, err := os.ReadDir(f.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != node.StoreFile && entry.Name() != node.StoreFile+"-wal" && entry.Name() != node.StoreFile+"-shm" {
			t.Fatalf("unexpected file %q in the store directory", entry.Name())
		}
	}
	ctl := f.Site.AddController(fabrictest.ControllerOptions{})
	pase, err := ctl.PASE(ctx, dev)
	if err != nil {
		t.Fatalf("PASE in an open window: %v", err)
	}
	if pase.Commissioned() {
		t.Fatal("PASE alone reports commissioned")
	}
	if closed := dev.Node().Sessions().ClosePASESessions(); closed != 1 {
		t.Fatalf("PASE sessions closed %d, want 1", closed)
	}
	pair, err := ctl.Commission(ctx, dev, fabrictest.CommissionOptions{})
	if err != nil || !pair.Commissioned() {
		t.Fatalf("commission: %v", err)
	}
	if got := nodetest.RootNumbers(bridge, "zone-a")["zone-a"]; got != 2 {
		t.Fatalf("first resource number %d, want 2", got)
	}
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v != 2100 {
		t.Fatalf("MeasuredValue %d null=%v", v, null)
	}
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMin); null || v != -5000 {
		t.Fatalf("MinMeasuredValue %d null=%v", v, null)
	}
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMax); null || v != 15000 {
		t.Fatalf("MaxMeasuredValue %d null=%v", v, null)
	}
	revision, err := fabrictest.Read(ctx, pair, client.AttributeAt(2, nodetest.ClusterTemperature, 0xFFFD))
	if err != nil || revision.Value.El.Uint == 0 {
		t.Fatalf("ClusterRevision %d, %v", revision.Value.El.Uint, err)
	}
	agg := nodetest.Descriptor(ctx, t, f.Site, pair, 1)
	if !slices.Equal(agg.DeviceTypeList, []descriptordef.DeviceTypeStruct{{DeviceType: 0x000E, Revision: 2}}) || !slices.Equal(agg.PartsList, []uint16{2}) {
		t.Fatalf("Aggregator descriptor %+v", agg)
	}
	zoneDescriptor := nodetest.Descriptor(ctx, t, f.Site, pair, 2)
	types := []uint32{}
	for _, d := range zoneDescriptor.DeviceTypeList {
		types = append(types, d.DeviceType)
	}
	slices.Sort(types)
	if !slices.Equal(types, []uint32{0x0013, 0x0302}) {
		t.Fatalf("zone device types %v", types)
	}
	if got := dev.Node().Bridge().Topology().FindByID(2).SourceKey.String(); got != "zone-a" {
		t.Fatalf("stable key %q", got)
	}
}

func TestRestartKeepsFabricAndNumbers(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	cut := []node.Resource{zone("zone-a", 2100), zone("zone-b", 2200), assembly("assembly-a", 1900, 2000)}
	f.Apply(ctx, cut...)
	dev, pair := f.Commission(ctx)
	before := f.Bridge.Numbers()
	fabrics, err := dev.Node().FabricInfos(ctx)
	if err != nil || len(fabrics) != 1 {
		t.Fatalf("fabrics %v, %v", fabrics, err)
	}
	if err := f.Bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	f.Start(ctx)
	f.Apply(ctx, cut...)
	after := f.Bridge.Numbers()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("numbers differ across a restart\nbefore %v\nafter  %v", before, after)
	}
	restarted, _ := f.Launcher.Last()
	fabricsAfter, err := restarted.Node().FabricInfos(ctx)
	if err != nil || len(fabricsAfter) != 1 || fabricsAfter[0].FabricID != fabrics[0].FabricID || fabricsAfter[0].NodeID != fabrics[0].NodeID {
		t.Fatalf("fabrics after restart %v, %v", fabricsAfter, err)
	}
	for key, number := range after {
		if key == "zone-a" || key == "zone-b" {
			if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, number, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v == 0 {
				t.Fatalf("%s at %d after restart: %d null=%v", key, number, v, null)
			}
		}
	}
}

func TestChangedValueReachesSubscriptionAndUnchangedDoesNot(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, zone("zone-a", 2100))
	_, pair := f.Commission(ctx)
	var mu sync.Mutex
	var reports []int64
	path := client.AttributeAt(2, nodetest.ClusterTemperature, nodetest.AttrMeasured)
	ceiling := time.Minute
	sub, err := fabrictest.Subscribe(ctx, pair, fabrictest.SubscribeRequest{
		Attributes: []fabrictest.AttributePath{path}, MaxIntervalCeiling: &ceiling,
		Updated: func(r *fabrictest.Report) error {
			mu.Lock()
			defer mu.Unlock()
			for _, a := range r.Attributes {
				if a.Path.Attribute == nodetest.AttrMeasured {
					reports = append(reports, a.Value.El.Int)
				}
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sub.Close)
	count := func() []int64 {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(reports)
	}
	if got := count(); !slices.Equal(got, []int64{2100}) {
		t.Fatalf("priming report %v", got)
	}
	f.Apply(ctx, zone("zone-a", 2150))
	if err := f.Site.Eventually(ctx, "changed value report", func() bool { return len(count()) == 2 }); err != nil {
		t.Fatal(err)
	}
	if got := count(); !slices.Equal(got, []int64{2100, 2150}) {
		t.Fatalf("reports %v", got)
	}
	f.Apply(ctx, zone("zone-a", 2150))
	if err := f.Site.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	if got := count(); len(got) != 2 {
		t.Fatalf("an unchanged value sent a report: %v", got)
	}
	if v, _ := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); v != 2150 {
		t.Fatalf("read %d", v)
	}
}

func TestUnknownValueBecomesNullWhereNullable(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	unknown := zone("zone-a", 0)
	unknown.Values = nil
	f.Apply(ctx, unknown)
	_, pair := f.Commission(ctx)
	if _, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); !null {
		t.Fatal("unknown value is not null")
	}
	f.Apply(ctx, zone("zone-a", 2000))
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v != 2000 {
		t.Fatalf("value after known %d null=%v", v, null)
	}
	f.Apply(ctx, unknown)
	if _, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); !null {
		t.Fatal("value that became unknown is not null")
	}
}

func TestMoreThanSixtyFourResourcesHaveDistinctStableNumbers(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	const count = 70
	cut := make([]node.Resource, 0, count)
	ids := make([]string, 0, count)
	for i := range count {
		id := "zone-" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		ids = append(ids, id)
		cut = append(cut, zone(id, 2000+i))
	}
	report := f.Apply(ctx, cut...)
	if len(report.Endpoints) != count || len(report.Refused) != 0 {
		t.Fatalf("endpoints %d refused %v", len(report.Endpoints), report.Refused)
	}
	first := nodetest.RootNumbers(f.Bridge, ids...)
	seen := map[uint16]string{}
	for id, n := range first {
		if other, ok := seen[n]; ok || n < 2 {
			t.Fatalf("%s and %s share number %d", id, other, n)
		}
		seen[n] = id
	}
	if len(seen) != count {
		t.Fatalf("%d distinct numbers", len(seen))
	}
	for i, id := range ids {
		if first[id] != uint16(2+i) {
			t.Fatalf("%s got %d, want %d", id, first[id], 2+i)
		}
	}
	if err := f.Bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	f.Start(ctx)
	f.Apply(ctx, cut...)
	if second := nodetest.RootNumbers(f.Bridge, ids...); !reflect.DeepEqual(first, second) {
		t.Fatalf("numbers moved across a restart")
	}
}

func TestWithdrawalKeepsNumberReservedAndReturnGetsItBack(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, zone("zone-a", 2100), zone("zone-b", 2200))
	dev, pair := f.Commission(ctx)
	if got := nodetest.RootNumbers(f.Bridge, "zone-a", "zone-b"); got["zone-a"] != 2 || got["zone-b"] != 3 {
		t.Fatalf("numbers %v", got)
	}
	f.Apply(ctx, zone("zone-b", 2200))
	if _, ok := f.Bridge.Numbers()["zone-a"]; ok {
		t.Fatal("withdrawn resource still has an endpoint")
	}
	if parts := nodetest.Descriptor(ctx, t, f.Site, pair, 1).PartsList; slices.Contains(parts, 2) {
		t.Fatalf("withdrawn endpoint is in the Aggregator PartsList %v", parts)
	}
	f.Apply(ctx, zone("zone-b", 2200), zone("zone-c", 2300))
	if got := nodetest.RootNumbers(f.Bridge, "zone-b", "zone-c"); got["zone-b"] != 3 || got["zone-c"] != 4 {
		t.Fatalf("the intervening resource took a reserved number: %v", got)
	}
	f.Apply(ctx, zone("zone-a", 2100), zone("zone-b", 2200), zone("zone-c", 2300))
	if got := nodetest.RootNumbers(f.Bridge, "zone-a", "zone-b", "zone-c"); got["zone-a"] != 2 || got["zone-b"] != 3 || got["zone-c"] != 4 {
		t.Fatalf("numbers after return %v", got)
	}
	if v, _ := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); v != 2100 {
		t.Fatalf("returned endpoint reads %d", v)
	}
	_ = dev
}

func TestWithdrawalReservationSurvivesRestart(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, zone("zone-a", 2100), zone("zone-b", 2200))
	f.Apply(ctx, zone("zone-b", 2200))
	if err := f.Bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	f.Start(ctx)
	f.Apply(ctx, zone("zone-b", 2200), zone("zone-c", 2300))
	if got := nodetest.RootNumbers(f.Bridge, "zone-b", "zone-c"); got["zone-b"] != 3 || got["zone-c"] != 4 {
		t.Fatalf("numbers %v", got)
	}
	f.Apply(ctx, zone("zone-a", 2100), zone("zone-b", 2200), zone("zone-c", 2300))
	if got := nodetest.RootNumbers(f.Bridge, "zone-a"); got["zone-a"] != 2 {
		t.Fatalf("returned resource got %d", got["zone-a"])
	}
}

func TestExclusionServesOnlyTheAdmittedAttribute(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	both := zone("zone-a", 2100)
	both.Elements = []string{nodetest.ElementTemperature, nodetest.ElementTolerance}
	both.Values[node.Selector{ClusterID: nodetest.ClusterTemperature, AttributeID: nodetest.AttrTolerance}] = node.Value{Value: 7}
	unadmitted := node.Resource{
		ID: "zone-x", Kind: nodetest.KindUnmapped, Elements: []string{nodetest.ElementUnrostered},
		Values: map[node.Selector]node.Value{measured(): {Value: 1}},
	}
	report := f.Apply(ctx, both, unadmitted)
	if _, ok := f.Bridge.Numbers()["zone-x"]; ok {
		t.Fatal("a wholly unadmitted resource has an endpoint")
	}
	wantWithheld := []publication.Withholding{
		{ResourceID: "zone-a", Element: nodetest.ElementTolerance, Reason: publication.ReasonUnmapped},
		{ResourceID: "zone-x", Element: nodetest.ElementUnrostered, Reason: publication.ReasonUnmapped},
	}
	if !reflect.DeepEqual(report.Withheld, wantWithheld) {
		t.Fatalf("withheld %+v", report.Withheld)
	}
	_, pair := f.Commission(ctx)
	if _, err := fabrictest.Read(ctx, pair, client.AttributeAt(2, nodetest.ClusterTemperature, nodetest.AttrTolerance)); err == nil {
		t.Fatal("the unadmitted optional attribute is served")
	}
	var mu sync.Mutex
	var attrs []uint32
	ceiling := time.Minute
	sub, err := fabrictest.Subscribe(ctx, pair, fabrictest.SubscribeRequest{
		Attributes: []fabrictest.AttributePath{
			client.AttributeAt(2, nodetest.ClusterTemperature, nodetest.AttrMeasured),
			client.AttributeAt(2, nodetest.ClusterTemperature, nodetest.AttrTolerance),
		},
		MaxIntervalCeiling: &ceiling,
		Updated: func(r *fabrictest.Report) error {
			mu.Lock()
			defer mu.Unlock()
			for _, a := range r.Attributes {
				attrs = append(attrs, a.Path.Attribute)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sub.Close)
	mu.Lock()
	primed := len(attrs)
	mu.Unlock()
	both.Values = map[node.Selector]node.Value{
		measured(): {Value: 2300},
		{ClusterID: nodetest.ClusterTemperature, AttributeID: nodetest.AttrTolerance}: {Value: 9},
	}
	f.Apply(ctx, both, unadmitted)
	if err := f.Site.Eventually(ctx, "admitted report", func() bool { mu.Lock(); defer mu.Unlock(); return len(attrs) > primed }); err != nil {
		t.Fatal(err)
	}
	if err := f.Site.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, id := range attrs[primed:] {
		if id != nodetest.AttrMeasured {
			t.Fatalf("a report carried attribute 0x%04X", id)
		}
	}
}

func TestRefusalsNameTheMissingRows(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	incomplete := node.Resource{ID: "incomplete-a", Kind: nodetest.KindIncomplete, Elements: []string{nodetest.ElementOrphan}}
	unknownKind := node.Resource{ID: "other-a", Kind: "resource.synthetic.nothing", Elements: []string{nodetest.ElementTemperature}}
	report := f.Apply(ctx, incomplete, unknownKind, zone("zone-a", 1))
	want := []node.Refusal{
		{ResourceID: "incomplete-a", Reason: "mandatory rows have neither a transformed element nor a declared value", Missing: []string{"Temperature Measurement.MinMeasuredValue"}},
		{ResourceID: "other-a", Reason: "no composition profile for kind resource.synthetic.nothing"},
	}
	if !reflect.DeepEqual(report.Refused, want) {
		t.Fatalf("refused %+v", report.Refused)
	}
	numbers := f.Bridge.Numbers()
	if _, ok := numbers["incomplete-a"]; ok || numbers["zone-a"] != 2 {
		t.Fatalf("numbers %v", numbers)
	}
}

func TestPartsGetStableKeysFromResourceAndRole(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, assembly("assembly-a", 1900, 2000), assembly("assembly-b", 1800, 1700))
	_, pair := f.Commission(ctx)
	numbers := f.Bridge.Numbers()
	keys := []string{
		node.RootKey("assembly-a").String(), node.PartKey("assembly-a", nodetest.PartSensor).String(), node.PartKey("assembly-a", nodetest.PartBackup).String(),
		node.RootKey("assembly-b").String(), node.PartKey("assembly-b", nodetest.PartSensor).String(), node.PartKey("assembly-b", nodetest.PartBackup).String(),
	}
	seen := map[uint16]string{}
	for _, k := range keys {
		n, ok := numbers[k]
		if !ok {
			t.Fatalf("no endpoint for key %q in %v", k, numbers)
		}
		if other, dup := seen[n]; dup {
			t.Fatalf("keys %q and %q share %d", k, other, n)
		}
		seen[n] = k
	}
	root := numbers[node.RootKey("assembly-a").String()]
	sensor := numbers[node.PartKey("assembly-a", nodetest.PartSensor).String()]
	backup := numbers[node.PartKey("assembly-a", nodetest.PartBackup).String()]
	if parts := nodetest.Descriptor(ctx, t, f.Site, pair, root).PartsList; !slices.Contains(parts, sensor) || !slices.Contains(parts, backup) {
		t.Fatalf("root parts %v lack %d and %d", parts, sensor, backup)
	}
	if v, _ := nodetest.ReadInt(ctx, t, f.Site, pair, sensor, nodetest.ClusterTemperature, nodetest.AttrMeasured); v != 1900 {
		t.Fatalf("sensor part %d", v)
	}
	if v, _ := nodetest.ReadInt(ctx, t, f.Site, pair, backup, nodetest.ClusterTemperature, nodetest.AttrMeasured); v != 2000 {
		t.Fatalf("backup part %d", v)
	}
	if v, _ := nodetest.ReadInt(ctx, t, f.Site, pair, numbers[node.PartKey("assembly-b", nodetest.PartSensor).String()], nodetest.ClusterTemperature, nodetest.AttrMeasured); v != 1800 {
		t.Fatalf("other assembly sensor %d", v)
	}
}

func TestCompositionReport(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	report := f.Apply(ctx, assembly("assembly-a", 1900, 2000), zone("zone-a", 2100))
	want := []node.EndpointReport{
		{
			ResourceIDs: []string{"assembly-a"}, Kind: nodetest.KindAssembly, DeviceType: 0x0013,
			Elements: []string{nodetest.ElementBackupValue, nodetest.ElementPartValue},
			Parts: []node.PartReport{
				{OwnerResourceID: "assembly-a", Role: nodetest.PartSensor, Key: node.PartKey("assembly-a", nodetest.PartSensor).String(), DeviceType: 0x0302,
					Rows: []string{"Temperature Measurement.MeasuredValue", "Temperature Measurement.MinMeasuredValue", "Temperature Measurement.MaxMeasuredValue"}},
				{OwnerResourceID: "assembly-a", Role: nodetest.PartBackup, Key: node.PartKey("assembly-a", nodetest.PartBackup).String(), DeviceType: 0x0302,
					Rows: []string{"Temperature Measurement.MeasuredValue", "Temperature Measurement.MinMeasuredValue", "Temperature Measurement.MaxMeasuredValue"}},
			},
		},
		{
			ResourceIDs: []string{"zone-a"}, RelationKeys: []string{"contains:zone-a"}, Kind: nodetest.KindZone, DeviceType: 0x0302,
			Elements: []string{nodetest.ElementTemperature},
		},
	}
	if !reflect.DeepEqual(report.Endpoints, want) {
		t.Fatalf("composition report\n got %+v\nwant %+v", report.Endpoints, want)
	}
}

func TestTwoLifecyclesAndPermutedOrderGiveTheSameNumbers(t *testing.T) {
	cut := []node.Resource{zone("zone-c", 3), assembly("assembly-a", 1, 2), zone("zone-a", 1), zone("zone-b", 2)}
	run := func(order []int) (map[string]uint16, node.Report) {
		f := newFixture(t)
		ctx := bounded(t)
		f.Start(ctx)
		permuted := make([]node.Resource, 0, len(cut))
		for _, i := range order {
			permuted = append(permuted, cut[i])
		}
		report := f.Apply(ctx, permuted...)
		return f.Bridge.Numbers(), report
	}
	numbersA, reportA := run([]int{0, 1, 2, 3})
	numbersB, reportB := run([]int{3, 2, 1, 0})
	if !reflect.DeepEqual(numbersA, numbersB) || !reflect.DeepEqual(reportA, reportB) {
		t.Fatalf("permuted order changed the result\n%v\n%v", numbersA, numbersB)
	}
	keys := make([]string, 0, len(numbersA))
	for k := range numbersA {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) != 6 {
		t.Fatalf("keys %q", keys)
	}
	f := newFixture(t)
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, cut...)
	first, firstReport := f.Bridge.Numbers(), f.Bridge.Report()
	if err := f.Bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	f.Start(ctx)
	f.Apply(ctx, cut...)
	if second := f.Bridge.Numbers(); !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstReport, f.Bridge.Report()) {
		t.Fatalf("a second lifecycle changed numbers or compositions")
	}
}

func TestConfigurationIsValidated(t *testing.T) {
	good := nodetest.Config(t.TempDir())
	cases := map[string]func(*node.Config){
		"empty directory":    func(c *node.Config) { c.StoreDir = "" },
		"other vendor":       func(c *node.Config) { c.ProductDescription.VendorID = 0xFFF2 },
		"zero product":       func(c *node.Config) { c.ProductDescription.ProductID = 0 },
		"duplicate kind":     func(c *node.Config) { c.Profiles = append(c.Profiles, c.Profiles[0]) },
		"repeated part role": func(c *node.Config) { c.Profiles[1].Parts[1].Role = nodetest.PartSensor },
		"empty part role":    func(c *node.Config) { c.Profiles[1].Parts[0].Role = "" },
		"duplicate roster":   func(c *node.Config) { c.Roster = append(c.Roster, c.Roster[0]) },
	}
	for name, mutate := range cases {
		cfg := good
		cfg.Profiles = nodetest.Profiles()
		cfg.Roster = nodetest.Roster()
		mutate(&cfg)
		if _, err := node.New(cfg); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := node.New(good); err != nil {
		t.Fatal(err)
	}
}

func TestApplyBeforeStartAndDuplicateIDs(t *testing.T) {
	f := newFixture(t)
	bridge, err := node.NewWith(nodetest.Config(f.Dir), f.Launcher)
	if err != nil {
		t.Fatal(err)
	}
	ctx := bounded(t)
	if _, err := bridge.Apply(ctx, nil); err == nil {
		t.Fatal("apply before start")
	}
	f.Bridge = bridge
	if err := bridge.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := bridge.Start(ctx); err == nil {
		t.Fatal("second start")
	}
	if _, err := bridge.Apply(ctx, []node.Resource{zone("zone-a", 1), zone("zone-a", 2)}); err == nil {
		t.Fatal("duplicate resource id")
	}
}

func TestRepeatedPartIsRefusedWithoutASourceBackedRole(t *testing.T) {
	cfg := nodetest.Config(t.TempDir())
	cfg.Profiles = nodetest.Profiles()
	cfg.Profiles[1].Parts[0].Count = 2
	site := fabrictest.NewSite(t, fabrictest.Options{})
	bridge, err := node.NewWith(cfg, &nodetest.SiteLauncher{Site: site})
	if err != nil {
		t.Fatal(err)
	}
	ctx := bounded(t)
	if err := bridge.Start(ctx); err != nil {
		t.Fatal(err)
	}
	report, err := bridge.Apply(ctx, []node.Resource{assembly("assembly-a", 1, 2)})
	if err != nil || len(report.Refused) != 1 || report.Refused[0].ResourceID != "assembly-a" || len(bridge.Numbers()) != 0 {
		t.Fatalf("report %+v, %v, numbers %v", report, err, bridge.Numbers())
	}
}

func TestDefaultLauncherRefusesAnInvalidNodeConfiguration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fresh")
	cfg := nodetest.Config(dir)
	cfg.Commissioning.Passcode = 0
	bridge, err := node.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := bridge.Start(bounded(t)); err == nil {
		t.Fatal("start with an invalid passcode")
	}
	if bridge.Node() != nil {
		t.Fatal("a failed start left a node")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the store directory was not created: %v", err)
	}
}

func TestCommissionAndReadOnTheLoopbackInterface(t *testing.T) {
	site := fabrictest.NewSite(t, fabrictest.Options{Loopback: true})
	f := &nodetest.Fixture{T: t, Site: site, Launcher: &nodetest.SiteLauncher{Site: site}, Dir: t.TempDir()}
	ctx := bounded(t)
	f.Start(ctx)
	f.Apply(ctx, zone("zone-a", 2100))
	dev, pair := f.Commission(ctx)
	if !dev.Addr().IP.IsLoopback() {
		t.Fatalf("device address %v is not on the loopback interface", dev.Addr())
	}
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v != 2100 {
		t.Fatalf("value %d null=%v", v, null)
	}
}
