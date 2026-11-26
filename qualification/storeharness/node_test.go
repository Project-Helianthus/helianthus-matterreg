//go:build gofabric

package storeharness_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	descriptordef "thingfabric.dev/go-fabric/cluster/spec/descriptor"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
)

// TestBridgeNodeStoreScenarios runs the durable-store scenarios against the
// real bridge node and its SDK store.
func TestBridgeNodeStoreScenarios(t *testing.T) {
	t.Run("more than 64 resources keep distinct numbers across a restart", func(t *testing.T) {
		f := nodetest.NewFixture(t)
		ctx := nodetest.Bounded(t)
		f.Start(ctx)
		const count = 80
		cut := make([]node.Resource, 0, count)
		ids := make([]string, 0, count)
		for i := range count {
			id := fmt.Sprintf("zone-%03d", i)
			ids = append(ids, id)
			cut = append(cut, nodetest.Zone(id, 2000+i))
		}
		f.Apply(ctx, cut...)
		first := nodetest.RootNumbers(f.Bridge, ids...)
		distinct := map[uint16]bool{}
		for _, n := range first {
			distinct[n] = true
		}
		if len(distinct) != count {
			t.Fatalf("%d distinct numbers for %d resources", len(distinct), count)
		}
		if err := f.Bridge.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		f.Start(ctx)
		f.Apply(ctx, cut...)
		if got := nodetest.RootNumbers(f.Bridge, ids...); !reflect.DeepEqual(got, first) {
			t.Fatal("numbers moved across a restart")
		}
	})

	t.Run("withdrawal keeps the number and the return gets it back", func(t *testing.T) {
		f := nodetest.NewFixture(t)
		ctx := nodetest.Bounded(t)
		f.Start(ctx)
		f.Apply(ctx, nodetest.Zone("zone-a", 1), nodetest.Zone("zone-b", 2))
		_, pair := f.Commission(ctx)
		if got := nodetest.RootNumbers(f.Bridge, "zone-a"); got["zone-a"] != 2 {
			t.Fatalf("first resource at %d, want 2", got["zone-a"])
		}
		agg := nodetest.Descriptor(ctx, t, f.Site, pair, 1)
		if !slices.Equal(agg.DeviceTypeList, []descriptordef.DeviceTypeStruct{{DeviceType: 0x000E, Revision: 2}}) {
			t.Fatalf("Aggregator at 1 is not readable: %+v", agg)
		}
		f.Apply(ctx, nodetest.Zone("zone-b", 2))
		f.Apply(ctx, nodetest.Zone("zone-b", 2), nodetest.Zone("zone-c", 3))
		got := nodetest.RootNumbers(f.Bridge, "zone-b", "zone-c")
		if got["zone-b"] != 3 || got["zone-c"] != 4 {
			t.Fatalf("intervening resource numbers %v", got)
		}
		f.Apply(ctx, nodetest.Zone("zone-a", 1), nodetest.Zone("zone-b", 2), nodetest.Zone("zone-c", 3))
		if back := nodetest.RootNumbers(f.Bridge, "zone-a"); back["zone-a"] != 2 {
			t.Fatalf("returned resource at %d, want 2", back["zone-a"])
		}
	})

	t.Run("two lifecycles and permuted input order give one key-to-number map", func(t *testing.T) {
		cut := []node.Resource{nodetest.Zone("zone-c", 3), nodetest.Assembly("assembly-a", 1, 2), nodetest.Zone("zone-a", 1), nodetest.Zone("zone-b", 2)}
		order := func(idx ...int) map[string]uint16 {
			f := nodetest.NewFixture(t)
			ctx := nodetest.Bounded(t)
			f.Start(ctx)
			var permuted []node.Resource
			for _, i := range idx {
				permuted = append(permuted, cut[i])
			}
			f.Apply(ctx, permuted...)
			return f.Bridge.Numbers()
		}
		forward, reverse := order(0, 1, 2, 3), order(3, 2, 1, 0)
		if !reflect.DeepEqual(forward, reverse) {
			t.Fatalf("permuted order changed the initial allocation\n%v\n%v", forward, reverse)
		}
		f := nodetest.NewFixture(t)
		ctx := nodetest.Bounded(t)
		f.Start(ctx)
		f.Apply(ctx, cut...)
		first := f.Bridge.Numbers()
		if err := f.Bridge.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		f.Start(ctx)
		f.Apply(ctx, cut[3], cut[2], cut[1], cut[0])
		if second := f.Bridge.Numbers(); !reflect.DeepEqual(first, second) {
			t.Fatalf("a second lifecycle changed the numbers\n%v\n%v", first, second)
		}
	})
}
