package nodetest

import (
	"context"
	"testing"
	"time"

	descriptordef "thingfabric.dev/go-fabric/cluster/spec/descriptor"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/fabrictest"

	bridgenode "github.com/Project-Helianthus/helianthus-matterreg/node"
)

// Bounded returns a context that ends with the test.
func Bounded(t testing.TB) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// Measured selects the root MeasuredValue of a zone.
func Measured() bridgenode.Selector {
	return bridgenode.Selector{ClusterID: ClusterTemperature, AttributeID: AttrMeasured}
}

// Zone is a published temperature resource with one measured value.
func Zone(id string, value int) bridgenode.Resource {
	return bridgenode.Resource{
		ID: id, Kind: KindZone, Elements: []string{ElementTemperature},
		RelationKeys: []string{"contains:" + id},
		Values:       map[bridgenode.Selector]bridgenode.Value{Measured(): {Value: value}},
	}
}

// Assembly is a published resource with two protocol parts.
func Assembly(id string, main, backup int) bridgenode.Resource {
	return bridgenode.Resource{
		ID: id, Kind: KindAssembly, Elements: []string{ElementPartValue, ElementBackupValue},
		Values: map[bridgenode.Selector]bridgenode.Value{
			{PartRole: PartSensor, ClusterID: ClusterTemperature, AttributeID: AttrMeasured}: {Value: main},
			{PartRole: PartBackup, ClusterID: ClusterTemperature, AttributeID: AttrMeasured}: {Value: backup},
		},
	}
}

// Fixture is one fabrictest site, one store directory and the bridges started
// on them in turn.
type Fixture struct {
	T        testing.TB
	Site     *fabrictest.Site
	Launcher *SiteLauncher
	Dir      string
	Bridge   *bridgenode.Bridge
	// Input, when set, is the SemanticInput of the next bridge Start builds.
	Input *Input
}

// NewFixture builds a site and a store directory.
func NewFixture(t testing.TB) *Fixture {
	t.Helper()
	site := fabrictest.NewSite(t, fabrictest.Options{})
	return &Fixture{T: t, Site: site, Launcher: &SiteLauncher{Site: site}, Dir: t.TempDir()}
}

// Start builds and starts a bridge on the directory.
func (f *Fixture) Start(ctx context.Context) *bridgenode.Bridge {
	f.T.Helper()
	bridge, err := bridgenode.NewWith(f.config(), f.Launcher)
	if err != nil {
		f.T.Fatal(err)
	}
	if err := bridge.Start(ctx); err != nil {
		f.T.Fatal(err)
	}
	f.Bridge = bridge
	return bridge
}

// Apply applies a cut and fails the test on an error.
func (f *Fixture) Apply(ctx context.Context, resources ...bridgenode.Resource) bridgenode.Report {
	f.T.Helper()
	report, err := f.Bridge.Apply(ctx, resources)
	if err != nil {
		f.T.Fatalf("apply: %v", err)
	}
	return report
}

// Commission commissions the latest device with a new controller.
func (f *Fixture) Commission(ctx context.Context) (*fabrictest.Device, *fabrictest.Pair) {
	f.T.Helper()
	dev, err := f.Launcher.Last()
	if err != nil {
		f.T.Fatal(err)
	}
	ctl := f.Site.AddController(fabrictest.ControllerOptions{})
	pair, err := ctl.Commission(ctx, dev, fabrictest.CommissionOptions{})
	if err != nil {
		f.T.Fatalf("commission: %v", err)
	}
	return dev, pair
}

// ReadAttribute reads one integer attribute over the controller's client node,
// which opens a new CASE session when the device restarted. null reports the
// null value.
func ReadAttribute(ctx context.Context, site *fabrictest.Site, pair *fabrictest.Pair, ep uint16, cluster, attr uint32) (value int64, null bool, err error) {
	n, err := pair.Node()
	if err != nil {
		return 0, false, err
	}
	err = site.Run(ctx, func(ctx context.Context) error {
		var req client.ReadRequest
		req.Attributes = []client.AttributePath{client.AttributeAt(ep, cluster, attr)}
		rep, err := n.Read(ctx, req)
		if err != nil {
			return err
		}
		if len(rep.Attributes) == 0 {
			return client.ErrNoValue
		}
		value, null = rep.Attributes[0].Value.El.Int, rep.Attributes[0].Value.El.IsNull
		return nil
	})
	return value, null, err
}

// ReadInt reads an integer attribute; null reports the null value.
func ReadInt(ctx context.Context, t testing.TB, site *fabrictest.Site, pair *fabrictest.Pair, ep uint16, cluster, attr uint32) (value int64, null bool) {
	t.Helper()
	value, null, err := ReadAttribute(ctx, site, pair, ep, cluster, attr)
	if err != nil {
		t.Fatalf("read %d/0x%04X/0x%04X: %v", ep, cluster, attr, err)
	}
	return value, null
}

// Descriptor reads the Descriptor cluster of an endpoint.
func Descriptor(ctx context.Context, t testing.TB, site *fabrictest.Site, pair *fabrictest.Pair, ep uint16) descriptordef.State {
	t.Helper()
	n, err := pair.Node()
	if err != nil {
		t.Fatal(err)
	}
	var state descriptordef.State
	err = site.Run(ctx, func(ctx context.Context) error {
		var req client.ReadRequest
		req.Attributes = []client.AttributePath{client.ClusterAt(ep, descriptordef.ClusterID)}
		rep, err := n.Read(ctx, req)
		if err != nil {
			return err
		}
		state, _ = descriptordef.StateFrom(rep.Values(ep, descriptordef.ClusterID))
		return nil
	})
	if err != nil {
		t.Fatalf("descriptor %d: %v", ep, err)
	}
	return state
}

// RootNumbers returns the endpoint number of each resource's root.
func RootNumbers(b *bridgenode.Bridge, ids ...string) map[string]uint16 {
	all := b.Numbers()
	out := map[string]uint16{}
	for _, id := range ids {
		out[id] = all[bridgenode.RootKey(id).String()]
	}
	return out
}

func (f *Fixture) config() bridgenode.Config {
	cfg := Config(f.Dir)
	if f.Input != nil {
		cfg.Input = f.Input
	}
	return cfg
}
