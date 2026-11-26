//go:build gofabric

package commissioning_test

import (
	"reflect"
	"slices"
	"testing"

	descriptordef "thingfabric.dev/go-fabric/cluster/spec/descriptor"
	"thingfabric.dev/go-fabric/fabrictest"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
)

// TestBridgeNodeCommissionsAndSurvivesARestart drives the real bridge node with
// the SDK's in-process controller: PASE in an open window, CommissioningComplete
// over CASE, the mandatory attributes of a bridged endpoint, and a restart on
// the same store directory that keeps the fabric and the endpoint numbers.
func TestBridgeNodeCommissionsAndSurvivesARestart(t *testing.T) {
	f := nodetest.NewFixture(t)
	ctx := nodetest.Bounded(t)
	f.Start(ctx)
	cut := []node.Resource{nodetest.Zone("zone-a", 2100), nodetest.Zone("zone-b", 2200)}
	f.Apply(ctx, cut...)
	dev, err := f.Launcher.Last()
	if err != nil {
		t.Fatal(err)
	}
	if !dev.Node().Window().Open {
		t.Fatal("no commissioning window is open")
	}
	ctl := f.Site.AddController(fabrictest.ControllerOptions{})
	pase, err := ctl.PASE(ctx, dev)
	if err != nil || pase.Commissioned() {
		t.Fatalf("PASE in an open window: commissioned=%v, %v", pase != nil && pase.Commissioned(), err)
	}
	if closed := dev.Node().Sessions().ClosePASESessions(); closed != 1 {
		t.Fatalf("PASE sessions closed %d", closed)
	}
	pair, err := ctl.Commission(ctx, dev, fabrictest.CommissionOptions{})
	if err != nil || !pair.Commissioned() {
		t.Fatalf("commission: %v", err)
	}
	numbers := nodetest.RootNumbers(f.Bridge, "zone-a", "zone-b")
	if numbers["zone-a"] != 2 || numbers["zone-b"] != 3 {
		t.Fatalf("numbers %v", numbers)
	}
	for id, want := range map[string][3]int64{"zone-a": {2100, -5000, 15000}, "zone-b": {2200, -5000, 15000}} {
		for i, attr := range []uint32{nodetest.AttrMeasured, nodetest.AttrMin, nodetest.AttrMax} {
			got, null := nodetest.ReadInt(ctx, t, f.Site, pair, numbers[id], nodetest.ClusterTemperature, attr)
			if null || got != want[i] {
				t.Fatalf("%s attribute 0x%04X = %d null=%v, want %d", id, attr, got, null, want[i])
			}
		}
	}
	fabrics, err := dev.Node().FabricInfos(ctx)
	if err != nil || len(fabrics) != 1 {
		t.Fatalf("fabrics %v, %v", fabrics, err)
	}

	if err := f.Bridge.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	f.Start(ctx)
	f.Apply(ctx, cut...)
	restarted, _ := f.Launcher.Last()
	after, err := restarted.Node().FabricInfos(ctx)
	if err != nil || len(after) != 1 || after[0].FabricID != fabrics[0].FabricID || after[0].NodeID != fabrics[0].NodeID {
		t.Fatalf("fabrics after the restart %v, %v", after, err)
	}
	if got := nodetest.RootNumbers(f.Bridge, "zone-a", "zone-b"); !reflect.DeepEqual(got, numbers) {
		t.Fatalf("numbers after the restart %v, want %v", got, numbers)
	}
	if got, null := nodetest.ReadInt(ctx, t, f.Site, pair, 3, nodetest.ClusterTemperature, nodetest.AttrMeasured); null || got != 2200 {
		t.Fatalf("read over the new CASE session %d null=%v", got, null)
	}
	if agg := nodetest.Descriptor(ctx, t, f.Site, pair, 1); !slices.Equal(agg.DeviceTypeList, []descriptordef.DeviceTypeStruct{{DeviceType: 0x000E, Revision: 2}}) {
		t.Fatalf("Aggregator %+v", agg)
	}
}
