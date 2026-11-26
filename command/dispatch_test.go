package command_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/graph"

	"github.com/Project-Helianthus/helianthus-matterreg/command"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	operationID  = "thermal.operation.set_temperature"
	capabilityID = "thermal.capability.set_temperature"
)

var node = driver.NodeRef{Bridge: "bridge.synthetic", ID: "zone-a"}

var row = composition.CommandRow{
	ClusterID: 0x0201, ElementID: 0x0012, Operation: operationID, Capability: capabilityID,
	InverseEncoder: command.InverseCentiCelsiusToKelvin,
}

type input struct {
	cut   driver.SemanticCut
	after func()
}

func (i *input) Capture(context.Context, driver.Filter) (driver.SemanticCut, error) {
	if i.after != nil {
		i.after()
	}
	return i.cut, nil
}

func (*input) Resume(context.Context, driver.Cursor, driver.Filter) (driver.Changes, error) {
	return driver.Changes{}, errors.New("not used")
}

func (*input) RetrieveLens(context.Context, graph.CutRef, graph.LensRequest) (graph.LensResult, error) {
	return graph.LensResult{}, errors.New("not used")
}

func (*input) SumMembers(context.Context, graph.CutRef, graph.LensRequest, semreg.DefinitionRef, semreg.ResourceID) (graph.MemberSum, error) {
	return graph.MemberSum{}, errors.New("not used")
}

type call struct {
	bound    bool
	request  driver.OperationRequest
	deadline time.Time
	has      bool
	ctxErr   error
}

type ops struct {
	mu     sync.Mutex
	calls  []call
	result driver.OperationResult
	err    error
}

func (o *ops) record(ctx context.Context, bound bool, r driver.OperationRequest) (driver.OperationResult, error) {
	deadline, has := ctx.Deadline()
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls = append(o.calls, call{bound: bound, request: r, deadline: deadline, has: has, ctxErr: ctx.Err()})
	return o.result, o.err
}

func (o *ops) Invoke(ctx context.Context, r driver.OperationRequest) (driver.OperationResult, error) {
	return o.record(ctx, false, r)
}

func (o *ops) InvokeCallBound(ctx context.Context, r driver.OperationRequest) (driver.OperationResult, error) {
	return o.record(ctx, true, r)
}

func ptr[T any](v T) *T { return &v }

func cut() driver.SemanticCut {
	pack := semreg.PackRef{ID: "helianthus.pack.thermal", Version: "1.0.0"}
	service := semreg.ServiceInstance{
		InstanceID: "service.a", AssetID: "asset.a", BindingID: "binding.a", SourceEpochID: "epoch.a",
		DriverGeneration: "2", Resource: ptr(semreg.ResourceID("zone-a")), Revision: "3",
	}
	capability := semreg.CapabilityInstance{
		InstanceID: "capability.a", AssetID: "asset.a", ServiceInstance: "service.a",
		Definition: semreg.DefinitionRef{Pack: pack, ID: capabilityID, Version: "1.0.0"},
		BindingID:  "binding.a", SourceEpochID: "epoch.a", DriverGeneration: "2", Revision: "4",
	}
	route := driver.SettingRoute{
		For:       driver.Element{Subject: node, Key: semreg.FactKey{FactID: "thermal.setpoint.temperature_setpoint"}},
		Operation: semreg.DefinitionRef{Pack: pack, ID: operationID, Version: "1.0.0"},
	}
	route.Route.CapabilityInstance, route.Route.ServiceInstance = "capability.a", "service.a"
	route.Route.BindingID, route.Route.SourceEpochID, route.Route.DriverGeneration = "binding.a", "epoch.a", "2"
	return driver.SemanticCut{
		Snapshots: []semreg.Snapshot{{
			AssetID: "asset.a", Revisions: semreg.RevisionVector{Semantic: "7", Capabilities: "5"},
			Services: []semreg.ServiceInstance{service}, Capabilities: []semreg.CapabilityInstance{capability},
		}},
		SettingRoutes: []driver.SettingRoute{route},
	}
}

func confirmed() driver.OperationResult {
	return driver.OperationResult{Status: command.StatusReadbackConfirmed}
}

func TestConfirmedWriteSucceedsAndFillsTheRequestFromTheCut(t *testing.T) {
	o := &ops{result: confirmed()}
	if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), row, int16(2100)); err != nil {
		t.Fatal(err)
	}
	if len(o.calls) != 1 || !o.calls[0].bound {
		t.Fatalf("calls %+v", o.calls)
	}
	r := o.calls[0].request
	want := driver.OperationRequest{
		AssetID: "asset.a", BindingID: "binding.a", CapabilityInstanceID: "capability.a", Operation: cut().SettingRoutes[0].Operation,
		ExpectedSemanticRevision: "7", ExpectedCapabilityRevision: "5", ExpectedCapabilityInstanceRevision: "4",
		ExpectedDriverGeneration: "2", ExpectedSourceEpochID: "epoch.a",
	}
	if r.AssetID != want.AssetID || r.BindingID != want.BindingID || r.CapabilityInstanceID != want.CapabilityInstanceID ||
		r.Operation != want.Operation || r.ExpectedSemanticRevision != want.ExpectedSemanticRevision ||
		r.ExpectedCapabilityRevision != want.ExpectedCapabilityRevision || r.ExpectedCapabilityInstanceRevision != want.ExpectedCapabilityInstanceRevision ||
		r.ExpectedDriverGeneration != want.ExpectedDriverGeneration || r.ExpectedSourceEpochID != want.ExpectedSourceEpochID {
		t.Fatalf("request %+v", r)
	}
	if r.Preconditions == nil || len(r.Preconditions) != 0 || r.Timeout != 0 {
		t.Fatalf("preconditions %v timeout %v", r.Preconditions, r.Timeout)
	}
}

// Every field of the route identity is varied alone. Each mutation of the
// production match that stops comparing the field makes its case send a call.
func TestEachFieldOfTheRouteIdentityIsRequired(t *testing.T) {
	cases := map[string]struct {
		field string
		edit  func(*driver.SemanticCut)
	}{
		"route subject":            {"route.For.Subject (resource ref)", func(c *driver.SemanticCut) { c.SettingRoutes[0].For.Subject.ID = "zone-b" }},
		"route subject bridge":     {"route.For.Subject.Bridge", func(c *driver.SemanticCut) { c.SettingRoutes[0].For.Subject.Bridge = "bridge.other" }},
		"route operation":          {"route.Operation.ID", func(c *driver.SemanticCut) { c.SettingRoutes[0].Operation.ID = "thermal.operation.other" }},
		"route capability":         {"route.Route.CapabilityInstance", func(c *driver.SemanticCut) { c.SettingRoutes[0].Route.CapabilityInstance = "capability.b" }},
		"route service":            {"route.Route.ServiceInstance", func(c *driver.SemanticCut) { c.SettingRoutes[0].Route.ServiceInstance = "service.b" }},
		"route binding":            {"route.Route.BindingID", func(c *driver.SemanticCut) { c.SettingRoutes[0].Route.BindingID = "binding.b" }},
		"route epoch":              {"route.Route.SourceEpochID", func(c *driver.SemanticCut) { c.SettingRoutes[0].Route.SourceEpochID = "epoch.b" }},
		"route generation":         {"route.Route.DriverGeneration", func(c *driver.SemanticCut) { c.SettingRoutes[0].Route.DriverGeneration = "3" }},
		"capability definition":    {"capability.Definition.ID", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].Definition.ID = "thermal.capability.read" }},
		"capability asset":         {"capability.AssetID", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].AssetID = "asset.b" }},
		"capability service":       {"capability.ServiceInstance", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].ServiceInstance = "service.b" }},
		"capability binding":       {"capability.BindingID", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].BindingID = "binding.b" }},
		"capability epoch":         {"capability.SourceEpochID", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].SourceEpochID = "epoch.b" }},
		"capability generation":    {"capability.DriverGeneration", func(c *driver.SemanticCut) { c.Snapshots[0].Capabilities[0].DriverGeneration = "1" }},
		"service resource":         {"service.Resource", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].Resource = ptr(semreg.ResourceID("zone-b")) }},
		"service without resource": {"service.Resource nil", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].Resource = nil }},
		"service asset":            {"service.AssetID", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].AssetID = "asset.b" }},
		"service binding":          {"service.BindingID", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].BindingID = "binding.b" }},
		"service epoch":            {"service.SourceEpochID", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].SourceEpochID = "epoch.b" }},
		"service generation":       {"service.DriverGeneration", func(c *driver.SemanticCut) { c.Snapshots[0].Services[0].DriverGeneration = "1" }},
		"no route":                 {"route list empty", func(c *driver.SemanticCut) { c.SettingRoutes = nil }},
		"two routes":               {"route list holds the route twice", func(c *driver.SemanticCut) { c.SettingRoutes = append(c.SettingRoutes, c.SettingRoutes[0]) }},
		"no snapshot":              {"snapshot list empty", func(c *driver.SemanticCut) { c.Snapshots = nil }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := cut()
			tc.edit(&c)
			o := &ops{result: confirmed()}
			err := command.New(node, &input{cut: c}, o).Write(context.Background(), row, int16(2100))
			if !errors.Is(err, command.ErrNoRoute) {
				t.Fatalf("%s: %v", tc.field, err)
			}
			if len(o.calls) != 0 {
				t.Fatalf("%s: %d calls", tc.field, len(o.calls))
			}
		})
	}
}

func TestRowFieldsOfOperationAndCapabilityAreRequired(t *testing.T) {
	for name, edit := range map[string]func(*composition.CommandRow){
		"operation":  func(r *composition.CommandRow) { r.Operation = "thermal.operation.other" },
		"capability": func(r *composition.CommandRow) { r.Capability = "thermal.capability.other" },
	} {
		t.Run(name, func(t *testing.T) {
			r := row
			edit(&r)
			o := &ops{result: confirmed()}
			if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), r, int16(2100)); !errors.Is(err, command.ErrNoRoute) || len(o.calls) != 0 {
				t.Fatalf("%v, %d calls", err, len(o.calls))
			}
		})
	}
}

func TestOnlyReadbackConfirmedIsSuccess(t *testing.T) {
	for _, status := range []driver.OperationStatus{command.StatusAccepted, command.StatusProtocolAck, command.StatusFailed, command.StatusUnknown, "UNAVAILABLE", ""} {
		o := &ops{result: driver.OperationResult{Status: status}}
		err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), row, int16(2100))
		if !errors.Is(err, command.ErrNotConfirmed) {
			t.Fatalf("status %q: %v", status, err)
		}
	}
	o := &ops{result: confirmed(), err: errors.New("router failure")}
	if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), row, int16(2100)); err == nil {
		t.Fatal("an error from the call was success")
	}
}

func TestAbsoluteDeadlineUsesTheFiniteEntryWithTheRemainingBudget(t *testing.T) {
	for _, budget := range []time.Duration{20 * time.Millisecond, 72 * time.Hour} {
		o := &ops{result: confirmed()}
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		err := command.New(node, &input{cut: cut()}, o).Write(ctx, row, int16(2100))
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		c := o.calls[0]
		if c.bound || !c.has {
			t.Fatalf("entry %+v", c)
		}
		if c.request.Timeout <= 0 || c.request.Timeout > budget || budget-c.request.Timeout > 5*time.Second {
			t.Fatalf("budget %v passed as %v", budget, c.request.Timeout)
		}
	}
}

func TestExpiredDeadlineIsPassedUnchangedToTheFiniteEntry(t *testing.T) {
	o := &ops{result: driver.OperationResult{Status: command.StatusFailed}}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := command.New(node, &input{cut: cut()}, o).Write(ctx, row, int16(2100)); err == nil {
		t.Fatal("a refused call was success")
	}
	if len(o.calls) != 1 || o.calls[0].bound || o.calls[0].request.Timeout > 0 {
		t.Fatalf("calls %+v", o.calls)
	}
}

func TestCallBoundChildIsCancelledOnReturnAndFollowsTheCallback(t *testing.T) {
	var seen context.Context
	o := &ops{result: confirmed()}
	probe := &probeOps{ops: o, seen: &seen}
	if err := command.New(node, &input{cut: cut()}, probe).Write(context.Background(), row, int16(2100)); err != nil {
		t.Fatal(err)
	}
	if seen.Err() == nil {
		t.Fatal("the child of the callback was not cancelled on return")
	}
	parent, cancel := context.WithCancel(context.Background())
	in := &input{cut: cut(), after: cancel}
	o = &ops{result: driver.OperationResult{Status: command.StatusFailed}}
	err := command.New(node, in, o).Write(parent, row, int16(2100))
	if err == nil || len(o.calls) != 1 || !errors.Is(o.calls[0].ctxErr, context.Canceled) {
		t.Fatalf("cancellation before the call: err %v calls %+v", err, o.calls)
	}
}

func TestCancellationAfterAPossibleSendIsNotSuccessAndIsNotReplayed(t *testing.T) {
	o := &ops{result: driver.OperationResult{Status: command.StatusUnknown}, err: context.Canceled}
	err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), row, int16(2100))
	if err == nil {
		t.Fatalf("err %v", err)
	}
	if len(o.calls) != 1 {
		t.Fatalf("%d calls", len(o.calls))
	}
}

type probeOps struct {
	*ops
	seen *context.Context
}

func (p *probeOps) InvokeCallBound(ctx context.Context, r driver.OperationRequest) (driver.OperationResult, error) {
	*p.seen = ctx
	return p.ops.InvokeCallBound(ctx, r)
}

func TestNothingIsCalledWithoutPortsOrInverseOrValue(t *testing.T) {
	o := &ops{result: confirmed()}
	if err := command.New(node, nil, o).Write(context.Background(), row, int16(1)); !errors.Is(err, command.ErrNoPorts) {
		t.Fatal(err)
	}
	if err := command.New(node, &input{cut: cut()}, nil).Write(context.Background(), row, int16(1)); !errors.Is(err, command.ErrNoPorts) {
		t.Fatal(err)
	}
	var none *command.Dispatcher
	if err := none.Write(context.Background(), row, int16(1)); !errors.Is(err, command.ErrNoPorts) {
		t.Fatal(err)
	}
	unknown := row
	unknown.InverseEncoder = "no_such_inverse"
	if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), unknown, int16(1)); !errors.Is(err, command.ErrNoInverse) {
		t.Fatal(err)
	}
	for _, value := range []any{nil, "text", struct{ A, B int8 }{1, 2}, struct{}{}, (*int16)(nil), uint64(1 << 63)} {
		if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), row, value); err == nil {
			t.Fatalf("value %#v was dispatched", value)
		}
	}
	if len(o.calls) != 0 {
		t.Fatalf("%d calls", len(o.calls))
	}
}

func TestInversesAreExactDecimals(t *testing.T) {
	cases := []struct {
		inverse, argument, unit, coefficient string
		exponent                             int32
		value                                any
	}{
		{command.InverseCentiCelsiusToKelvin, "thermal.setpoint.temperature_setpoint", "unit.kelvin", "29415", -2, int16(2100)},
		{command.InverseCentiCelsiusToKelvin, "thermal.setpoint.temperature_setpoint", "unit.kelvin", "27315", -2, int16(0)},
		{command.InverseCentiCelsiusToKelvin, "thermal.setpoint.temperature_setpoint", "unit.kelvin", "26315", -2, int16(-1000)},
		{command.InverseMilliampereToAmpere, "evse.limit.allocated_current", "unit.ampere", "16000", -3, int64(16000)},
		{command.InverseMilliampereToAmpere, "evse.limit.allocated_current", "unit.ampere", "6000", -3, struct{ MaximumChargeCurrent uint16 }{6000}},
	}
	for _, tc := range cases {
		o := &ops{result: confirmed()}
		r := row
		r.InverseEncoder = tc.inverse
		if err := command.New(node, &input{cut: cut()}, o).Write(context.Background(), r, tc.value); err != nil {
			t.Fatal(err)
		}
		argument := o.calls[0].request.Arguments[0]
		quantity := argument.Value.Quantity
		if string(argument.ID) != tc.argument || argument.Value.Kind != semreg.ValueQuantity || string(quantity.Unit) != tc.unit ||
			quantity.Number.Coefficient != tc.coefficient || quantity.Number.Exponent10 != tc.exponent {
			t.Fatalf("%s %v: %+v", tc.inverse, tc.value, argument)
		}
	}
}

// The session names no bridge for a mounted resource, so the mount carries
// none and the route is matched by the resource id; a second route of the same
// resource id under another bridge is a second match and gives no write.
func TestAMountWithoutABridgeMatchesTheRouteByResourceID(t *testing.T) {
	bare := driver.NodeRef{ID: node.ID}
	o := &ops{result: confirmed()}
	if err := command.New(bare, &input{cut: cut()}, o).Write(context.Background(), row, int16(2100)); err != nil {
		t.Fatal(err)
	}
	if len(o.calls) != 1 {
		t.Fatalf("calls %d", len(o.calls))
	}
	twice := cut()
	other := twice.SettingRoutes[0]
	other.For.Subject.Bridge = "bridge.other"
	twice.SettingRoutes = append(twice.SettingRoutes, other)
	o = &ops{result: confirmed()}
	if err := command.New(bare, &input{cut: twice}, o).Write(context.Background(), row, int16(2100)); !errors.Is(err, command.ErrNoRoute) || len(o.calls) != 0 {
		t.Fatalf("%v, %d calls", err, len(o.calls))
	}
	elsewhere := cut()
	elsewhere.SettingRoutes[0].For.Subject.ID = "zone-b"
	o = &ops{result: confirmed()}
	if err := command.New(bare, &input{cut: elsewhere}, o).Write(context.Background(), row, int16(2100)); !errors.Is(err, command.ErrNoRoute) || len(o.calls) != 0 {
		t.Fatalf("%v, %d calls", err, len(o.calls))
	}
}
