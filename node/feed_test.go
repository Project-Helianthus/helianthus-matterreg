package node_test

import (
	"reflect"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/fabrictest"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

const dimension = "synthetic.dimension"

func zoneItem(id string, kelvin *string) nodetest.Item {
	item := nodetest.Item{ID: id, Kind: nodetest.KindZone}
	fact := nodetest.Fact{Element: nodetest.ElementTemperature, Dimension: dimension}
	if kelvin != nil {
		fact.Value = nodetest.Kelvin(*kelvin, -2)
	}
	item.Facts = []nodetest.Fact{fact}
	return item
}

func text(s string) *string { return &s }

func TestRefreshWithoutAnInputIsRefused(t *testing.T) {
	f := newFixture(t)
	ctx := bounded(t)
	bridge := f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != node.ErrNoInput {
		t.Fatalf("refresh without an input: %v", err)
	}
}

func TestThermalZoneInKelvinReachesTheControllerAsCentiCelsius(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29415"))}, nil)
	bridge := f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	_, pair := f.Commission(ctx)
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v != 2100 {
		t.Fatalf("294.15 K reads %d null=%v, want 2100", v, null)
	}
	if f.Input.Captures != 1 || f.Input.Retrieved != 1 {
		t.Fatalf("captures %d lens reads %d", f.Input.Captures, f.Input.Retrieved)
	}
}

func TestRefreshReportsChangedValueAndUnchangedIsSilent(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29415"))}, nil)
	bridge := f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	_, pair := f.Commission(ctx)
	var mu sync.Mutex
	var reports []int64
	ceiling := time.Minute
	sub, err := fabrictest.Subscribe(ctx, pair, fabrictest.SubscribeRequest{
		Attributes:         []fabrictest.AttributePath{client.AttributeAt(2, nodetest.ClusterTemperature, nodetest.AttrMeasured)},
		MaxIntervalCeiling: &ceiling,
		Updated: func(r *fabrictest.Report) error {
			mu.Lock()
			defer mu.Unlock()
			for _, a := range r.Attributes {
				reports = append(reports, a.Value.El.Int)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sub.Close)
	count := func() []int64 { mu.Lock(); defer mu.Unlock(); return slices.Clone(reports) }
	if got := count(); !slices.Equal(got, []int64{2100}) {
		t.Fatalf("priming %v", got)
	}
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29515"))}, nil)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.Site.Eventually(ctx, "changed value report", func() bool { return len(count()) == 2 }); err != nil {
		t.Fatal(err)
	}
	if got := count(); !slices.Equal(got, []int64{2100, 2200}) {
		t.Fatalf("reports %v", got)
	}
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29515"))}, nil)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.Site.Settle(ctx); err != nil {
		t.Fatal(err)
	}
	if got := count(); len(got) != 2 {
		t.Fatalf("an unchanged value sent a report: %v", got)
	}
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", nil)}, nil)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if _, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); !null {
		t.Fatal("a value that became unknown is not null")
	}
}

func TestRefreshAfterRestartCapturesAgainAndKeepsNumbers(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	f.Input.Publish([]nodetest.Item{zoneItem("zone-b", text("29415")), zoneItem("zone-a", text("29415"))}, nil)
	bridge := f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	before := bridge.Numbers()
	if err := bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	bridge = f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, bridge.Numbers()) || f.Input.Captures != 2 {
		t.Fatalf("numbers %v then %v, captures %d", before, bridge.Numbers(), f.Input.Captures)
	}
}

func TestExclusionOverCASEThroughTheFeed(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	build := func(kelvin string, tolerance string) []nodetest.Item {
		both := zoneItem("zone-a", text(kelvin))
		both.Facts = append(both.Facts, nodetest.Fact{Element: nodetest.ElementTolerance, Value: nodetest.Kelvin(tolerance, 0), Dimension: dimension})
		unadmitted := nodetest.Item{ID: "zone-x", Kind: nodetest.KindUnmapped, Facts: []nodetest.Fact{{Element: nodetest.ElementUnrostered, Value: nodetest.Kelvin("29415", -2), Dimension: dimension}}}
		return []nodetest.Item{both, unadmitted}
	}
	f.Input.Publish(build("29415", "7"), nil)
	bridge := f.Start(ctx)
	report, err := bridge.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []publication.Withholding{
		{ResourceID: "zone-a", Element: nodetest.ElementTolerance, Reason: publication.ReasonUnmapped},
		{ResourceID: "zone-x", Element: nodetest.ElementUnrostered, Reason: publication.ReasonUnmapped},
	}
	if !reflect.DeepEqual(report.Withheld, want) {
		t.Fatalf("withheld %+v", report.Withheld)
	}
	if _, ok := bridge.Numbers()["zone-x"]; ok {
		t.Fatal("a wholly unadmitted resource has an endpoint")
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
	f.Input.Publish(build("29515", "9"), nil)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
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

func TestCompositionReportFromTheLensEqualsTheHandWrittenRecord(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	assembly := nodetest.Item{ID: "assembly-a", Kind: nodetest.KindAssembly, Facts: []nodetest.Fact{
		{Element: nodetest.ElementPartValue, Value: nodetest.Kelvin("29015", -2), Dimension: dimension},
		{Element: nodetest.ElementBackupValue, Value: nodetest.Kelvin("29115", -2), Dimension: dimension},
	}}
	zone := zoneItem("zone-a", text("29415"))
	zone.Component, zone.Depth = true, 1
	f.Input.Publish([]nodetest.Item{zone, assembly}, []nodetest.Relation{{Definition: "contains", From: "assembly-a", To: "zone-a"}})
	bridge := f.Start(ctx)
	report, err := bridge.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rows := []string{"Temperature Measurement.MeasuredValue", "Temperature Measurement.MinMeasuredValue", "Temperature Measurement.MaxMeasuredValue"}
	want := []node.EndpointReport{
		{
			ResourceIDs: []string{"assembly-a"}, RelationKeys: []string{"contains:assembly-a>zone-a"}, Kind: nodetest.KindAssembly, DeviceType: 0x0013,
			Elements: []string{nodetest.ElementBackupValue, nodetest.ElementPartValue},
			Parts: []node.PartReport{
				{OwnerResourceID: "assembly-a", Role: nodetest.PartSensor, Key: node.PartKey("assembly-a", nodetest.PartSensor).String(), DeviceType: 0x0302, Rows: rows},
				{OwnerResourceID: "assembly-a", Role: nodetest.PartBackup, Key: node.PartKey("assembly-a", nodetest.PartBackup).String(), DeviceType: 0x0302, Rows: rows},
			},
		},
		{
			ResourceIDs: []string{"zone-a"}, RelationKeys: []string{"contains:assembly-a>zone-a"}, Kind: nodetest.KindZone, DeviceType: 0x0302,
			Elements: []string{nodetest.ElementTemperature},
		},
	}
	if !reflect.DeepEqual(report.Endpoints, want) {
		t.Fatalf("composition report\n got %+v\nwant %+v", report.Endpoints, want)
	}
	numbers := bridge.Numbers()
	if numbers["assembly-a"] >= numbers["zone-a"] {
		t.Fatalf("an equipment resource is numbered after a component: %v", numbers)
	}
}

func TestShippedPilotIsWithheldWithItsMissingRowsNamed(t *testing.T) {
	source, err := composition.Load()
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	cfg := nodetest.Config(f.Dir)
	cfg.Input = f.Input
	cfg.Profiles = nil
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal("no pilot profile")
	}
	cfg.Profiles = []composition.Profile{profile}
	cfg.Roster = []publication.Row{{Element: "evse.ac.current", Kind: "attribute", Disposition: publication.Transformed}}
	var want []string
	for _, row := range pilot.Rows() {
		if row.Mandatory && row.Origin == pilot.OriginMissing && !(row.AttributeID >= 0xFFF8 && row.AttributeID < 0xFFFC) && row.AttributeID != 0xFFFD {
			name := row.ClusterName + "." + row.Name
			if row.Field != "" {
				name += "." + row.Field
			}
			want = append(want, name)
		}
	}
	sort.Strings(want)
	if len(want) == 0 {
		t.Fatal("the shipped pilot has no missing mandatory row")
	}
	f.Input.Publish([]nodetest.Item{{ID: "phase-a", Kind: pilot.ResourceKind, Facts: []nodetest.Fact{{Element: "evse.ac.current", Dimension: "evse.dimension.phase"}}}}, nil)
	ctx := bounded(t)
	bridge, err := node.NewWith(cfg, f.Launcher)
	if err != nil {
		t.Fatal(err)
	}
	if err := bridge.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bridge.Stop(ctx) })
	report, err := bridge.Refresh(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Endpoints) != 0 || len(bridge.Numbers()) != 0 {
		t.Fatalf("the shipped pilot published an endpoint: %+v", report.Endpoints)
	}
	if len(report.Refused) != 1 || report.Refused[0].ResourceID != "phase-a" || !reflect.DeepEqual(report.Refused[0].Missing, want) {
		t.Fatalf("refused %+v\nwant missing %v", report.Refused, want)
	}
}

func TestRefreshAfterResyncReadsCommitsAgainAndDoesNotRecapture(t *testing.T) {
	f := newFixture(t)
	f.Input = nodetest.NewInput()
	ctx := bounded(t)
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29415"))}, nil)
	bridge := f.Start(ctx)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	f.Input.Reincarnate()
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if f.Input.Captures != 2 {
		t.Fatalf("captures %d after a resync, want 2", f.Input.Captures)
	}
	f.Input.Publish([]nodetest.Item{zoneItem("zone-a", text("29515"))}, nil)
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if f.Input.Captures != 2 {
		t.Fatalf("captures %d, the bridge recaptured instead of resuming", f.Input.Captures)
	}
	_, pair := f.Commission(ctx)
	if v, null := nodetest.ReadInt(ctx, t, f.Site, pair, 2, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || v != 2200 {
		t.Fatalf("reads %d null=%v, want 2200", v, null)
	}
}
