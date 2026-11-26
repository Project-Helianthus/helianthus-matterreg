package node

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
	"thingfabric.dev/go-fabric/cluster/spec/windowcovering"
	"thingfabric.dev/go-fabric/controller/client"
	"thingfabric.dev/go-fabric/fabrictest"
	"thingfabric.dev/go-fabric/im"
	gfnode "thingfabric.dev/go-fabric/node"
	"thingfabric.dev/go-fabric/tlv"

	"github.com/Project-Helianthus/helianthus-matterreg/command"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	clusterThermostat        uint32 = 0x0201
	attrHeatingSetpoint      uint32 = 0x0012
	clusterWindowCovering    uint32 = 0x0102
	cmdGoToTilt              uint32 = 0x08
	kindWindowCovering              = "resource.synthetic.window-covering"
	cmdGoToLift              uint32 = 0x05
	kindThermostat                  = "resource.synthetic.thermostat"
	kindThermostatPair              = "resource.synthetic.thermostat-pair"
	elementSetpoint                 = "synthetic.thermostat.setpoint"
	elementSetpointBackup           = "synthetic.thermostat.backup-setpoint"
	operationSetTemperature         = "thermal.operation.set_temperature"
	capabilitySetTemperature        = "thermal.capability.set_temperature"
	bridgeID                        = "bridge.synthetic"
)

func descriptor(deviceType string, servers ...any) []composition.AttributeRow {
	return []composition.AttributeRow{
		{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0000, Name: "DeviceTypeList", MatterType: "list", Mandatory: true, Origin: composition.OriginDeclared, Declared: []any{map[string]any{"device_type": deviceType, "revision": 1}}},
		{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0001, Name: "ServerList", MatterType: "list", Mandatory: true, Origin: composition.OriginDeclared, Declared: servers},
	}
}

func thermostatRows(element string) []composition.AttributeRow {
	return []composition.AttributeRow{
		{ClusterID: clusterThermostat, ClusterName: "Thermostat", AttributeID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Mandatory: true, Origin: composition.OriginDeclared, Declared: 1},
		{ClusterID: clusterThermostat, ClusterName: "Thermostat", AttributeID: attrHeatingSetpoint, Name: "OccupiedHeatingSetpoint", MatterType: "int16", Mandatory: true, Origin: composition.OriginSemreg, Element: element, Encoder: "kelvin_to_centicelsius"},
	}
}

func setpointRow(part string) composition.CommandRow {
	return composition.CommandRow{
		PartKey: part, ClusterID: clusterThermostat, ElementID: attrHeatingSetpoint,
		Operation: operationSetTemperature, Capability: capabilitySetTemperature,
		InverseEncoder: command.InverseCentiCelsiusToKelvin,
	}
}

func typedProfiles() map[string]composition.Profile {
	thermostat := composition.Profile{
		Kind:       kindThermostat,
		Attributes: append(descriptor("0x0301", "0x001D", "0x0201"), thermostatRows(elementSetpoint)...),
		Commands:   []composition.CommandRow{setpointRow("")},
	}
	pair := composition.Profile{
		Kind:       kindThermostatPair,
		Attributes: descriptor("0x0013", "0x001D"),
		Parts: []composition.PartSpec{
			{Role: "main", DeviceTypeID: 0x0301, Revision: 7, Count: 1, Servers: []composition.ServerRow{{ClusterID: clusterThermostat, Name: "Thermostat"}}, Attributes: thermostatRows(elementSetpoint)},
			{Role: "backup", DeviceTypeID: 0x0301, Revision: 7, Count: 1, Servers: []composition.ServerRow{{ClusterID: clusterThermostat, Name: "Thermostat"}}, Attributes: thermostatRows(elementSetpointBackup)},
		},
		Commands: []composition.CommandRow{setpointRow("main")},
	}
	return map[string]composition.Profile{kindThermostat: thermostat, kindThermostatPair: pair}
}

// fakeInput is a synthetic SemanticInput whose Capture returns the cut the test
// holds at that moment; after, when set, runs once the cut is handed out.
type fakeInput struct {
	mu       sync.Mutex
	cut      driver.SemanticCut
	captures int
	after    func()
}

func (f *fakeInput) Capture(context.Context, driver.Filter) (driver.SemanticCut, error) {
	f.mu.Lock()
	f.captures++
	cut, after := f.cut, f.after
	f.mu.Unlock()
	if after != nil {
		after()
	}
	return cut, nil
}

func (f *fakeInput) set(cut driver.SemanticCut) {
	f.mu.Lock()
	f.cut = cut
	f.mu.Unlock()
}

func (f *fakeInput) current() driver.SemanticCut {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cut
}

func (f *fakeInput) Resume(context.Context, driver.Cursor, driver.Filter) (driver.Changes, error) {
	return driver.Changes{}, errors.New("not used")
}

func (f *fakeInput) RetrieveLens(context.Context, graph.CutRef, graph.LensRequest) (graph.LensResult, error) {
	return graph.LensResult{}, errors.New("not used")
}

func (f *fakeInput) SumMembers(context.Context, graph.CutRef, graph.LensRequest, semreg.DefinitionRef, semreg.ResourceID) (graph.MemberSum, error) {
	return graph.MemberSum{}, errors.New("not used")
}

type opCall struct {
	bound       bool
	request     driver.OperationRequest
	hasDeadline bool
	ctxErr      error
}

// fakeOps is a synthetic ScopedOperations. It admits a request only when the
// request names the route the input holds at that moment, as the router
// rechecks on its own cut; an admitted request is a send.
type fakeOps struct {
	mu     sync.Mutex
	input  *fakeInput
	calls  []opCall
	sends  int
	status driver.OperationStatus
	onSend func(driver.OperationRequest)
}

func (o *fakeOps) run(ctx context.Context, bound bool, r driver.OperationRequest) (driver.OperationResult, error) {
	_, has := ctx.Deadline()
	call := opCall{bound: bound, request: r, hasDeadline: has, ctxErr: ctx.Err()}
	admitted := call.ctxErr == nil && o.admits(r)
	o.mu.Lock()
	o.calls = append(o.calls, call)
	if admitted {
		o.sends++
	}
	status, onSend := o.status, o.onSend
	o.mu.Unlock()
	if !admitted {
		return driver.OperationResult{Status: command.StatusFailed}, nil
	}
	if onSend != nil {
		onSend(r)
	}
	return driver.OperationResult{Status: status}, nil
}

func (o *fakeOps) admits(r driver.OperationRequest) bool {
	cut := o.input.current()
	for _, route := range cut.SettingRoutes {
		for _, snapshot := range cut.Snapshots {
			if snapshot.AssetID != r.AssetID || route.Route.CapabilityInstance != r.CapabilityInstanceID ||
				route.Route.BindingID != r.BindingID || route.Route.SourceEpochID != r.ExpectedSourceEpochID ||
				route.Route.DriverGeneration != r.ExpectedDriverGeneration || route.Operation != r.Operation {
				continue
			}
			if snapshot.Revisions.Semantic == r.ExpectedSemanticRevision && snapshot.Revisions.Capabilities == r.ExpectedCapabilityRevision {
				return true
			}
		}
	}
	return false
}

func (o *fakeOps) Invoke(ctx context.Context, r driver.OperationRequest) (driver.OperationResult, error) {
	return o.run(ctx, false, r)
}

func (o *fakeOps) InvokeCallBound(ctx context.Context, r driver.OperationRequest) (driver.OperationResult, error) {
	return o.run(ctx, true, r)
}

func (o *fakeOps) snapshot() (calls []opCall, sends int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]opCall(nil), o.calls...), o.sends
}

func ptr[T any](v T) *T { return &v }

// cutFor builds one cut for resource: a snapshot with the service of that
// resource, the capability the row declares and its setting route. generation
// is the driver generation of all three.
func cutFor(resource, asset, generation string) driver.SemanticCut {
	nodeRef := driver.NodeRef{Bridge: bridgeID, ID: semreg.ResourceID(resource)}
	pack := semreg.PackRef{ID: "helianthus.pack.thermal", Version: "1.0.0"}
	operation := semreg.DefinitionRef{Pack: pack, ID: operationSetTemperature, Version: "1.0.0"}
	service := semreg.ServiceInstance{
		InstanceID: "service." + semreg.ServiceInstanceID(resource), AssetID: semreg.AssetID(asset),
		BindingID: "binding.synthetic", SourceEpochID: "epoch.synthetic", DriverGeneration: semreg.Uint64(generation),
		Resource: ptr(semreg.ResourceID(resource)), Revision: "3",
	}
	capability := semreg.CapabilityInstance{
		InstanceID: "capability." + semreg.CapabilityInstanceID(resource), AssetID: semreg.AssetID(asset),
		ServiceInstance: service.InstanceID, Definition: semreg.DefinitionRef{Pack: pack, ID: capabilitySetTemperature, Version: "1.0.0"},
		BindingID: "binding.synthetic", SourceEpochID: "epoch.synthetic", DriverGeneration: semreg.Uint64(generation), Revision: "4",
	}
	element := driver.Element{Subject: nodeRef, Key: semreg.FactKey{FactID: "thermal.setpoint.temperature_setpoint"}}
	route := driver.SettingRoute{For: element, Operation: operation}
	route.Route.CapabilityInstance, route.Route.ServiceInstance = capability.InstanceID, service.InstanceID
	route.Route.BindingID, route.Route.SourceEpochID, route.Route.DriverGeneration = capability.BindingID, capability.SourceEpochID, capability.DriverGeneration
	snapshot := semreg.Snapshot{
		AssetID: semreg.AssetID(asset), Revisions: semreg.RevisionVector{Semantic: "7", Capabilities: "5"},
		Services: []semreg.ServiceInstance{service}, Capabilities: []semreg.CapabilityInstance{capability},
	}
	return driver.SemanticCut{Snapshots: []semreg.Snapshot{snapshot}, SettingRoutes: []driver.SettingRoute{route}}
}

type rig struct {
	t      *testing.T
	site   *fabrictest.Site
	dev    *fabrictest.Device
	pair   *fabrictest.Pair
	input  *fakeInput
	ops    *fakeOps
	mounts map[string]*Mount
}

func newRig(t *testing.T) *rig {
	t.Helper()
	site := fabrictest.NewSite(t, fabrictest.Options{Loopback: true})
	dev := site.AddNode(gfnode.Config{
		ProductDescription: gfnode.ProductDescription{VendorID: DevelopmentVendorID, ProductID: ProductID},
		Commissioning: gfnode.Commissioning{
			Passcode: fabrictest.DefaultPasscode, Discriminator: fabrictest.DefaultDiscriminator,
			PBKDFIterations: fabrictest.DefaultPBKDFIterations, Salt: []byte("SPAKE2P Key Salt"),
		},
		BasicInformation: gfnode.BasicInformation{
			VendorName: "Synthetic", ProductName: "Synthetic bridge", NodeLabel: "bridge", SerialNumber: "SYN-1",
			HardwareVersion: 1, HardwareVersionString: "1", SoftwareVersion: 1, SoftwareVersionString: "1",
		},
	})
	input := &fakeInput{}
	return &rig{t: t, site: site, dev: dev, input: input, ops: &fakeOps{input: input, status: command.StatusReadbackConfirmed}, mounts: map[string]*Mount{}}
}

func (r *rig) ctx() context.Context {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	r.t.Cleanup(cancel)
	return ctx
}

// mount gives composeEndpoint the typed profile, the admitted elements and the
// session's ports, then adds the composed endpoint to the running node.
func (r *rig) mount(profile composition.Profile, resource string, values map[Selector]Value, elements ...string) *Mount {
	r.t.Helper()
	admitted := map[string]struct{}{}
	for _, element := range elements {
		admitted[element] = struct{}{}
	}
	rows, err := classify(profile, admitted)
	if err != nil {
		r.t.Fatal(err)
	}
	ports := &CommandPorts{Bridge: bridgeID, Input: r.input, Operations: r.ops}
	mount, err := composeEndpoint(resource, rows, values, commandRows(profile), ports)
	if err != nil {
		r.t.Fatal(err)
	}
	err = r.dev.Node().Aggregator().Batch(r.ctx(), func(b *gfnode.Batch) error { return b.Add(mount) })
	if err != nil {
		r.t.Fatal(err)
	}
	r.mounts[resource] = mount
	return mount
}

func (r *rig) commission() {
	r.t.Helper()
	pair, err := r.site.AddController(fabrictest.ControllerOptions{}).Commission(r.ctx(), r.dev, fabrictest.CommissionOptions{})
	if err != nil {
		r.t.Fatalf("commission: %v", err)
	}
	r.pair = pair
}

func (r *rig) endpoint(key string) uint16 {
	r.t.Helper()
	for _, ep := range r.dev.Node().Bridge().Topology().Endpoints {
		if ep != nil && ep.SourceKey != nil && ep.SourceKey.String() == key {
			return ep.ID
		}
	}
	r.t.Fatalf("no endpoint for %q", key)
	return 0
}

func (r *rig) read(endpoint uint16, cluster, attribute uint32) int64 {
	r.t.Helper()
	data, err := fabrictest.Read(r.ctx(), r.pair, client.AttributeAt(endpoint, cluster, attribute))
	if err != nil {
		r.t.Fatalf("read: %v", err)
	}
	return data.Value.El.Int
}

// write sends one write of the setpoint and returns the Interaction Model
// status the controller received. timeout, when positive, makes it timed.
func (r *rig) write(endpoint uint16, value int16, timed time.Duration, beforeFollowUp func()) im.StatusCode {
	r.t.Helper()
	n, err := r.pair.Node()
	if err != nil {
		r.t.Fatal(err)
	}
	var encodable spec.Encodable = spec.Int16(value)
	if beforeFollowUp != nil {
		encodable = hooked{Encodable: encodable, run: beforeFollowUp}
	}
	var statuses client.WriteStatuses
	err = r.site.Run(r.ctx(), func(ctx context.Context) error {
		var err error
		statuses, err = n.Write(ctx, []client.WriteRequest{{Path: client.AttributeAt(endpoint, clusterThermostat, attrHeatingSetpoint), Value: encodable}}, client.WriteOptions{TimedTimeout: timed})
		return err
	})
	var refused *client.StatusError
	if errors.As(err, &refused) {
		return refused.Status
	}
	if err != nil {
		r.t.Fatalf("write: %v", err)
	}
	return statuses[0].Status
}

type hooked struct {
	spec.Encodable
	run func()
}

func (h hooked) EncodeTLV(enc *tlv.Encoder, tag tlv.Tag) {
	h.run()
	h.Encodable.EncodeTLV(enc, tag)
}

func (r *rig) zone(resource string, initial int16) (endpoint uint16, mount *Mount) {
	r.t.Helper()
	r.input.set(cutFor(resource, "asset.synthetic.zone", "2"))
	profile := typedProfiles()[kindThermostat]
	mount = r.mount(profile, resource, map[Selector]Value{{ClusterID: clusterThermostat, AttributeID: attrHeatingSetpoint}: {Value: int(initial)}}, elementSetpoint)
	r.commission()
	return r.endpoint(resource), mount
}

func (r *rig) setpoint(endpoint uint16) int64 {
	return r.read(endpoint, clusterThermostat, attrHeatingSetpoint)
}

// TestManagedCommandMount runs on the ordinary build. composeEndpoint gets the
// typed profile and the session's ports; the real CASE listener on the loopback
// interface carries the controller's writes to the mounted servers.
func TestManagedCommandMount(t *testing.T) {
	t.Run("a confirmed write is stored and runs the call-bound entry", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		if got := r.setpoint(ep); got != 2000 {
			t.Fatalf("initial setpoint %d", got)
		}
		if status := r.write(ep, 2100, 0, nil); status != im.StatusSuccess {
			t.Fatalf("status %v", status)
		}
		calls, sends := r.ops.snapshot()
		if len(calls) != 1 || sends != 1 {
			t.Fatalf("calls %d sends %d", len(calls), sends)
		}
		call := calls[0]
		if !call.bound || call.hasDeadline || call.request.Timeout != 0 {
			t.Fatalf("scoped entry %+v", call)
		}
		if got := r.setpoint(ep); got != 2100 {
			t.Fatalf("setpoint %d after a confirmed write", got)
		}
		request := call.request
		if request.Operation.ID != operationSetTemperature || len(request.Arguments) != 1 || len(request.Preconditions) != 0 || request.Preconditions == nil {
			t.Fatalf("request %+v", request)
		}
		quantity := request.Arguments[0].Value.Quantity
		if request.Arguments[0].ID != "thermal.setpoint.temperature_setpoint" || quantity.Unit != "unit.kelvin" || quantity.Number.Coefficient != "29415" || quantity.Number.Exponent10 != -2 {
			t.Fatalf("argument %+v", request.Arguments[0])
		}
		if request.AssetID != "asset.synthetic.zone" || request.ExpectedSemanticRevision != "7" || request.ExpectedCapabilityRevision != "5" ||
			request.ExpectedCapabilityInstanceRevision != "4" || request.ExpectedDriverGeneration != "2" || request.ExpectedSourceEpochID != "epoch.synthetic" {
			t.Fatalf("request identity %+v", request)
		}
	})

	t.Run("the cut of each invocation is fresh", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		r.write(ep, 2100, 0, nil)
		r.input.set(cutFor("zone-a", "asset.synthetic.zone", "3"))
		r.write(ep, 2200, 0, nil)
		calls, _ := r.ops.snapshot()
		if len(calls) != 2 || calls[0].request.ExpectedDriverGeneration != "2" || calls[1].request.ExpectedDriverGeneration != "3" {
			t.Fatalf("calls %+v", calls)
		}
	})

	t.Run("the readback, not the wish, is what the attribute shows", func(t *testing.T) {
		r := newRig(t)
		ep, mount := r.zone("zone-a", 2000)
		if status := r.write(ep, 2100, 0, nil); status != im.StatusSuccess {
			t.Fatalf("status %v", status)
		}
		if got := r.setpoint(ep); got != 2100 {
			t.Fatalf("the SDK holds %d, not the written value, before a readback", got)
		}
		if err := mount.Servers[""][clusterThermostat].SetAttributes(map[uint32]any{attrHeatingSetpoint: int16(2050)}); err != nil {
			t.Fatal(err)
		}
		if got := r.setpoint(ep); got != 2050 {
			t.Fatalf("setpoint %d after the readback", got)
		}
	})

	for name, status := range map[string]driver.OperationStatus{
		"accepted": command.StatusAccepted, "protocol_ack": command.StatusProtocolAck,
		"failed": command.StatusFailed, "unknown": command.StatusUnknown,
	} {
		t.Run("status "+name+" answers FAILURE and stores nothing", func(t *testing.T) {
			r := newRig(t)
			ep, _ := r.zone("zone-a", 2000)
			r.ops.status = status
			if got := r.write(ep, 2100, 0, nil); got != im.StatusFailure {
				t.Fatalf("status %v", got)
			}
			if got := r.setpoint(ep); got != 2000 {
				t.Fatalf("setpoint %d", got)
			}
		})
	}

	t.Run("a route replaced between capture and admission sends nothing", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		r.input.after = func() { r.input.set(cutFor("zone-a", "asset.synthetic.zone", "9")) }
		if got := r.write(ep, 2100, 0, nil); got != im.StatusFailure {
			t.Fatalf("status %v", got)
		}
		calls, sends := r.ops.snapshot()
		if len(calls) != 1 || sends != 0 {
			t.Fatalf("calls %d sends %d", len(calls), sends)
		}
		if got := r.setpoint(ep); got != 2000 {
			t.Fatalf("setpoint %d", got)
		}
	})

	t.Run("a cut without exactly one matching route reaches no call", func(t *testing.T) {
		withdrawn := cutFor("zone-a", "asset.synthetic.zone", "2")
		withdrawn.SettingRoutes = nil
		doubled := cutFor("zone-a", "asset.synthetic.zone", "2")
		doubled.SettingRoutes = append(doubled.SettingRoutes, doubled.SettingRoutes[0])
		otherResource := cutFor("zone-b", "asset.synthetic.zone", "2")
		wrongCapability := cutFor("zone-a", "asset.synthetic.zone", "2")
		wrongCapability.Snapshots[0].Capabilities[0].Definition.ID = "thermal.capability.read.zone"
		staleService := cutFor("zone-a", "asset.synthetic.zone", "2")
		staleService.Snapshots[0].Services[0].DriverGeneration = "1"
		otherAsset := cutFor("zone-a", "asset.synthetic.zone", "2")
		otherAsset.Snapshots[0].Capabilities[0].AssetID = "asset.synthetic.other"
		for name, cut := range map[string]driver.SemanticCut{
			"no route": withdrawn, "two routes": doubled, "another resource": otherResource,
			"another capability": wrongCapability, "stale service": staleService, "another asset": otherAsset,
		} {
			t.Run(name, func(t *testing.T) {
				r := newRig(t)
				ep, _ := r.zone("zone-a", 2000)
				r.input.set(cut)
				if got := r.write(ep, 2100, 0, nil); got != im.StatusFailure {
					t.Fatalf("status %v", got)
				}
				if calls, _ := r.ops.snapshot(); len(calls) != 0 {
					t.Fatalf("calls %d", len(calls))
				}
				if got := r.setpoint(ep); got != 2000 {
					t.Fatalf("setpoint %d", got)
				}
			})
		}
	})

	t.Run("a sibling capability of the fixed resource is not read", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		cut := cutFor("zone-a", "asset.synthetic.zone", "2")
		stale := cutFor("zone-a", "asset.synthetic.zone", "1")
		stale.Snapshots[0].Capabilities[0].InstanceID = "capability.stale"
		stale.SettingRoutes[0].Route.CapabilityInstance = "capability.stale"
		cut.Snapshots[0].Capabilities = append(stale.Snapshots[0].Capabilities, cut.Snapshots[0].Capabilities...)
		r.input.set(cut)
		if got := r.write(ep, 2100, 0, nil); got != im.StatusSuccess {
			t.Fatalf("status %v", got)
		}
		calls, _ := r.ops.snapshot()
		if len(calls) != 1 || calls[0].request.CapabilityInstanceID != "capability.zone-a" || calls[0].request.ExpectedDriverGeneration != "2" {
			t.Fatalf("calls %+v", calls)
		}
	})

	t.Run("an attribute with no row is refused with no call", func(t *testing.T) {
		r := newRig(t)
		r.input.set(cutFor("assembly-a", "asset.synthetic.zone", "2"))
		profile := typedProfiles()[kindThermostatPair]
		r.mount(profile, "assembly-a", map[Selector]Value{
			{PartRole: "main", ClusterID: clusterThermostat, AttributeID: attrHeatingSetpoint}:   {Value: 2000},
			{PartRole: "backup", ClusterID: clusterThermostat, AttributeID: attrHeatingSetpoint}: {Value: 1900},
		}, elementSetpoint, elementSetpointBackup)
		r.commission()
		main, backup := r.endpoint(PartKey("assembly-a", "main").String()), r.endpoint(PartKey("assembly-a", "backup").String())
		if got := r.write(backup, 2100, 0, nil); got != im.StatusFailure {
			t.Fatalf("unmapped part status %v", got)
		}
		if calls, _ := r.ops.snapshot(); len(calls) != 0 {
			t.Fatalf("calls %d", len(calls))
		}
		if got := r.setpoint(backup); got != 1900 {
			t.Fatalf("backup setpoint %d", got)
		}
		if got := r.write(main, 2100, 0, nil); got != im.StatusSuccess {
			t.Fatalf("mapped part status %v", got)
		}
		if got := r.setpoint(main); got != 2100 {
			t.Fatalf("main setpoint %d", got)
		}
	})

	t.Run("a follow-up after the timed window is TIMEOUT with no call", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		status := r.write(ep, 2100, time.Millisecond, func() { r.site.Fake().Advance(time.Second) })
		if status != im.StatusTimeout {
			t.Fatalf("status %v", status)
		}
		if calls, _ := r.ops.snapshot(); len(calls) != 0 {
			t.Fatalf("calls %d", len(calls))
		}
		if got := r.setpoint(ep); got != 2000 {
			t.Fatalf("setpoint %d", got)
		}
	})

	t.Run("a mapped command is dispatched and answered from the correlated readback", func(t *testing.T) {
		r := newRig(t)
		r.input.set(cutFor("cover-a", "asset.synthetic.zone", "2"))
		r.mount(windowCoveringProfile(), "cover-a", nil)
		r.commission()
		ep := r.endpoint("cover-a")
		result, err := fabrictest.Invoke(r.ctx(), r.pair, client.CommandAt(ep, clusterWindowCovering, cmdGoToLift), windowcovering.GoToLiftPercentageRequest{LiftPercent100thsValue: 5000})
		if err != nil || result.Status != im.StatusSuccess {
			t.Fatalf("invoke %v %v", result.Status, err)
		}
		calls, sends := r.ops.snapshot()
		if len(calls) != 1 || sends != 1 || !calls[0].bound || calls[0].request.Timeout != 0 {
			t.Fatalf("calls %+v sends %d", calls, sends)
		}
		quantity := calls[0].request.Arguments[0].Value.Quantity
		if quantity.Number.Coefficient != "32315" || quantity.Number.Exponent10 != -2 {
			t.Fatalf("argument %+v", quantity)
		}
	})

	t.Run("a mapped command whose call is not confirmed answers FAILURE", func(t *testing.T) {
		for _, status := range []driver.OperationStatus{command.StatusAccepted, command.StatusProtocolAck, command.StatusFailed, command.StatusUnknown} {
			r := newRig(t)
			r.input.set(cutFor("cover-a", "asset.synthetic.zone", "2"))
			r.ops.status = status
			r.mount(windowCoveringProfile(), "cover-a", nil)
			r.commission()
			_, err := fabrictest.Invoke(r.ctx(), r.pair, client.CommandAt(r.endpoint("cover-a"), clusterWindowCovering, cmdGoToLift), windowcovering.GoToLiftPercentageRequest{LiftPercent100thsValue: 5000})
			var refused *client.StatusError
			if !errors.As(err, &refused) || refused.Status != im.StatusFailure {
				t.Fatalf("%s: %v", status, err)
			}
		}
	})

	t.Run("a command the server does not accept is UNSUPPORTED_COMMAND with no call", func(t *testing.T) {
		r := newRig(t)
		r.input.set(cutFor("cover-a", "asset.synthetic.zone", "2"))
		r.mount(windowCoveringProfile(), "cover-a", nil)
		r.commission()
		_, err := fabrictest.Invoke(r.ctx(), r.pair, client.CommandAt(r.endpoint("cover-a"), clusterWindowCovering, cmdGoToTilt), nil)
		var refused *client.StatusError
		if !errors.As(err, &refused) || refused.Status != im.StatusUnsupportedCommand {
			t.Fatalf("%v", err)
		}
		if calls, _ := r.ops.snapshot(); len(calls) != 0 {
			t.Fatalf("calls %d", len(calls))
		}
	})

	t.Run("a follow-up inside the timed window is dispatched", func(t *testing.T) {
		r := newRig(t)
		ep, _ := r.zone("zone-a", 2000)
		if status := r.write(ep, 2100, time.Minute, nil); status != im.StatusSuccess {
			t.Fatalf("status %v", status)
		}
		calls, _ := r.ops.snapshot()
		if len(calls) != 1 || !calls[0].bound || calls[0].request.Timeout != 0 {
			t.Fatalf("calls %+v", calls)
		}
	})
}

// windowCoveringProfile is the typed fixture of a mapped request command: the
// row is a test row, not a product mapping.
func windowCoveringProfile() composition.Profile {
	return composition.Profile{
		Kind: kindWindowCovering,
		Attributes: append(descriptor("0x0202", "0x001D", "0x0102"),
			composition.AttributeRow{ClusterID: clusterWindowCovering, ClusterName: "Window Covering", AttributeID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Mandatory: true, Origin: composition.OriginDeclared, Declared: 5}),
		Commands: []composition.CommandRow{{
			ClusterID: clusterWindowCovering, ElementID: cmdGoToLift, Operation: operationSetTemperature,
			Capability: capabilitySetTemperature, InverseEncoder: command.InverseCentiCelsiusToKelvin,
		}},
	}
}
