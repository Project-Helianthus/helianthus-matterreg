//go:build gofabric

package sdkfabric_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/graph"
	"thingfabric.dev/go-fabric/cluster/spec"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/fabrictest"
	"thingfabric.dev/go-fabric/im"
	"thingfabric.dev/go-fabric/node"

	"github.com/Project-Helianthus/helianthus-matterreg/command"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	thermostatCluster uint32 = 0x0201
	heatingSetpoint   uint32 = 0x0012
	setpointOperation        = "thermal.operation.set_temperature"
	setpointCapacity         = "thermal.capability.set_temperature"
)

type commandDeclaration struct{ decl spec.ClusterDeclaration }

func (c commandDeclaration) ClusterID() uint32 { return c.decl.Cluster }

func (c commandDeclaration) ClusterDeclaration() spec.ClusterDeclaration { return c.decl }

type commandInput struct{ cut driver.SemanticCut }

func (i commandInput) Capture(context.Context, driver.Filter) (driver.SemanticCut, error) {
	return i.cut, nil
}

func (commandInput) Resume(context.Context, driver.Cursor, driver.Filter) (driver.Changes, error) {
	return driver.Changes{}, errors.New("not used")
}

func (commandInput) RetrieveLens(context.Context, graph.CutRef, graph.LensRequest) (graph.LensResult, error) {
	return graph.LensResult{}, errors.New("not used")
}

func (commandInput) SumMembers(context.Context, graph.CutRef, graph.LensRequest, semreg.DefinitionRef, semreg.ResourceID) (graph.MemberSum, error) {
	return graph.MemberSum{}, errors.New("not used")
}

type commandOps struct {
	mu     sync.Mutex
	calls  int
	status driver.OperationStatus
}

func (o *commandOps) count(status driver.OperationStatus) (driver.OperationResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls++
	return driver.OperationResult{Status: status}, nil
}

func (o *commandOps) Invoke(context.Context, driver.OperationRequest) (driver.OperationResult, error) {
	return o.count(o.status)
}

func (o *commandOps) InvokeCallBound(context.Context, driver.OperationRequest) (driver.OperationResult, error) {
	return o.count(o.status)
}

func commandCut(resource string) driver.SemanticCut {
	pack := semreg.PackRef{ID: "helianthus.pack.thermal", Version: "1.0.0"}
	resourceID := semreg.ResourceID(resource)
	service := semreg.ServiceInstance{
		InstanceID: "service.reference", AssetID: "asset.reference", BindingID: "binding.reference", SourceEpochID: "epoch.reference",
		DriverGeneration: "1", Resource: &resourceID, Revision: "1",
	}
	capability := semreg.CapabilityInstance{
		InstanceID: "capability.reference", AssetID: "asset.reference", ServiceInstance: service.InstanceID,
		Definition: semreg.DefinitionRef{Pack: pack, ID: setpointCapacity, Version: "1.0.0"},
		BindingID:  "binding.reference", SourceEpochID: "epoch.reference", DriverGeneration: "1", Revision: "1",
	}
	route := driver.SettingRoute{
		For:       driver.Element{Subject: driver.NodeRef{Bridge: "bridge.reference", ID: resourceID}},
		Operation: semreg.DefinitionRef{Pack: pack, ID: setpointOperation, Version: "1.0.0"},
	}
	route.Route.CapabilityInstance, route.Route.ServiceInstance = capability.InstanceID, service.InstanceID
	route.Route.BindingID, route.Route.SourceEpochID, route.Route.DriverGeneration = capability.BindingID, capability.SourceEpochID, "1"
	return driver.SemanticCut{
		Snapshots: []semreg.Snapshot{{
			AssetID: "asset.reference", Revisions: semreg.RevisionVector{Semantic: "1", Capabilities: "1"},
			Services: []semreg.ServiceInstance{service}, Capabilities: []semreg.CapabilityInstance{capability},
		}},
		SettingRoutes: []driver.SettingRoute{route},
	}
}

// commandSite runs the ordinary dispatcher over a veto on a reference-site
// thermostat server, which the in-memory site's controller writes to.
func commandSite(t *testing.T, status driver.OperationStatus, cut driver.SemanticCut) (*fabrictest.Site, *fabrictest.Pair, uint16, *commandOps) {
	t.Helper()
	site := fabrictest.NewSite(t, fabrictest.Options{})
	if site.Net() == nil || site.Loopback() {
		t.Fatal("the site is not on the in-memory network")
	}
	dev := site.AddNode(node.Config{
		ProductDescription: node.ProductDescription{VendorID: 0xFFF1, ProductID: 0x8001},
		Commissioning: node.Commissioning{
			Passcode: fabrictest.DefaultPasscode, Discriminator: fabrictest.DefaultDiscriminator,
			PBKDFIterations: fabrictest.DefaultPBKDFIterations, Salt: []byte("SPAKE2P Key Salt"),
		},
		BasicInformation: node.BasicInformation{
			VendorName: "Synthetic", ProductName: "Synthetic", NodeLabel: "reference", SerialNumber: "SYN-1",
			HardwareVersion: 1, HardwareVersionString: "1", SoftwareVersion: 1, SoftwareVersionString: "1",
		},
	})
	deviceType, ok := device.Lookup(0x0301)
	if !ok {
		t.Fatal("no thermostat device type")
	}
	built, err := device.New(deviceType, device.Options{StableKey: device.StringKey("zone-a"), Name: "zone-a", DeviceAddress: "zone-a"},
		commandDeclaration{spec.ClusterDeclaration{Cluster: thermostatCluster, Features: 1, HasFeatures: true, Initial: map[uint32]any{heatingSetpoint: int16(2000)}}})
	if err != nil {
		t.Fatal(err)
	}
	ops := &commandOps{status: status}
	row := composition.CommandRow{
		ClusterID: thermostatCluster, ElementID: heatingSetpoint, Operation: setpointOperation,
		Capability: setpointCapacity, InverseEncoder: command.InverseCentiCelsiusToKelvin,
	}
	dispatcher := command.New(driver.NodeRef{Bridge: "bridge.reference", ID: "zone-a"}, commandInput{cut}, ops)
	veto, ok := built.Server(thermostatCluster).(spec.BeforeWriteSetter)
	if !ok {
		t.Fatal("the thermostat server takes no write veto")
	}
	veto.SetBeforeWrite(func(ctx context.Context, attribute uint32, value any) error {
		if attribute != heatingSetpoint {
			return command.ErrNoRoute
		}
		return dispatcher.Write(ctx, row, value)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	if err := dev.Node().Aggregator().Batch(ctx, func(b *node.Batch) error { return b.Add(built) }); err != nil {
		t.Fatal(err)
	}
	pair, err := site.AddController(fabrictest.ControllerOptions{}).Commission(ctx, dev, fabrictest.CommissionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ep := range dev.Node().Bridge().Topology().Endpoints {
		if ep != nil && ep.SourceKey != nil && ep.SourceKey.String() == "zone-a" {
			return site, pair, ep.ID, ops
		}
	}
	t.Fatal("no endpoint")
	return nil, nil, 0, nil
}

func writeSetpoint(t *testing.T, pair *fabrictest.Pair, ep uint16, value int16) error {
	t.Helper()
	return fabrictest.Write(context.Background(), pair, client.AttributeAt(ep, thermostatCluster, heatingSetpoint), spec.Int16(value))
}

func readSetpoint(t *testing.T, pair *fabrictest.Pair, ep uint16) int64 {
	t.Helper()
	data, err := fabrictest.Read(context.Background(), pair, client.AttributeAt(ep, thermostatCluster, heatingSetpoint))
	if err != nil {
		t.Fatal(err)
	}
	return data.Value.El.Int
}

func TestCommandDispatcherOnTheReferenceSite(t *testing.T) {
	t.Run("a confirmed write is stored", func(t *testing.T) {
		_, pair, ep, ops := commandSite(t, command.StatusReadbackConfirmed, commandCut("zone-a"))
		if err := writeSetpoint(t, pair, ep, 2100); err != nil {
			t.Fatal(err)
		}
		if got := readSetpoint(t, pair, ep); got != 2100 || ops.calls != 1 {
			t.Fatalf("setpoint %d calls %d", got, ops.calls)
		}
	})
	for _, status := range []driver.OperationStatus{command.StatusAccepted, command.StatusProtocolAck, command.StatusUnknown, command.StatusFailed} {
		t.Run("status "+string(status)+" is FAILURE and stores nothing", func(t *testing.T) {
			_, pair, ep, ops := commandSite(t, status, commandCut("zone-a"))
			var refused *client.StatusError
			if err := writeSetpoint(t, pair, ep, 2100); !errors.As(err, &refused) || refused.Status != im.StatusFailure {
				t.Fatalf("%v", err)
			}
			if got := readSetpoint(t, pair, ep); got != 2000 || ops.calls != 1 {
				t.Fatalf("setpoint %d calls %d", got, ops.calls)
			}
		})
	}
	t.Run("another resource's route sends nothing", func(t *testing.T) {
		_, pair, ep, ops := commandSite(t, command.StatusReadbackConfirmed, commandCut("zone-b"))
		var refused *client.StatusError
		if err := writeSetpoint(t, pair, ep, 2100); !errors.As(err, &refused) || refused.Status != im.StatusFailure {
			t.Fatalf("%v", err)
		}
		if got := readSetpoint(t, pair, ep); got != 2000 || ops.calls != 0 {
			t.Fatalf("setpoint %d calls %d", got, ops.calls)
		}
	})
}
