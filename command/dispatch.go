// Package command dispatches a Matter attribute write or request command of a
// bridged resource as the SemReg operation its profile row names, through the
// session's scoped operations. Success is the correlated readback and nothing
// else.
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/operation"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Operation status values of a SemReg call. SemReg states this closed set in a
// package a driver does not import.
const (
	StatusAccepted          driver.OperationStatus = "accepted"
	StatusProtocolAck       driver.OperationStatus = "protocol_ack"
	StatusReadbackConfirmed driver.OperationStatus = "readback_confirmed"
	StatusFailed            driver.OperationStatus = "failed"
	StatusUnknown           driver.OperationStatus = "unknown"
)

// Errors of dispatch. None carries an Interaction Model status, so the SDK
// answers FAILURE.
var (
	ErrNoPorts      = errors.New("command: no semantic input or scoped operations")
	ErrNoInverse    = errors.New("command: the row names no known inverse encoder")
	ErrNoRoute      = errors.New("command: the cut holds no single exactly scoped route for the row")
	ErrNotConfirmed = errors.New("command: the operation has no correlated readback")
)

// Dispatcher dispatches the commands of one mounted resource.
type Dispatcher struct {
	node  driver.NodeRef
	input driver.SemanticInput
	ops   driver.ScopedOperations
}

// New returns the dispatcher of the resource node. A nil input or operations
// dispatches nothing.
func New(node driver.NodeRef, input driver.SemanticInput, ops driver.ScopedOperations) *Dispatcher {
	return &Dispatcher{node: node, input: input, ops: ops}
}

// Write dispatches the write of value to the attribute the row maps.
func (d *Dispatcher) Write(ctx context.Context, row composition.CommandRow, value any) error {
	return d.dispatch(ctx, row, value)
}

// Invoke dispatches the request command the row maps with its decoded fields.
func (d *Dispatcher) Invoke(ctx context.Context, row composition.CommandRow, fields any) error {
	return d.dispatch(ctx, row, fields)
}

func (d *Dispatcher) dispatch(ctx context.Context, row composition.CommandRow, value any) error {
	if d == nil || d.input == nil || d.ops == nil {
		return ErrNoPorts
	}
	inv, ok := inverses[row.InverseEncoder]
	if !ok {
		return fmt.Errorf("%w: %q", ErrNoInverse, row.InverseEncoder)
	}
	matter, err := integerOf(value)
	if err != nil {
		return fmt.Errorf("command: %w", err)
	}
	cut, err := d.input.Capture(ctx, driver.Filter{})
	if err != nil {
		return fmt.Errorf("command: capture: %w", err)
	}
	request, err := d.request(cut, row)
	if err != nil {
		return err
	}
	request.Arguments = []semreg.TypedField{inv.field(matter)}
	var result driver.OperationResult
	if deadline, has := ctx.Deadline(); has {
		request.Timeout = time.Until(deadline)
		result, err = d.ops.Invoke(ctx, request)
	} else {
		bound, cancel := context.WithCancel(ctx)
		defer cancel()
		result, err = d.ops.InvokeCallBound(bound, request)
	}
	if err != nil {
		return fmt.Errorf("command: %w", err)
	}
	if result.Status != StatusReadbackConfirmed {
		return fmt.Errorf("%w: status %q", ErrNotConfirmed, result.Status)
	}
	return nil
}

// request resolves the one route of the row on the cut and fills the closed
// request from that cut's own identities and revisions.
func (d *Dispatcher) request(cut driver.SemanticCut, row composition.CommandRow) (driver.OperationRequest, error) {
	var found []driver.OperationRequest
	for _, route := range cut.SettingRoutes {
		if !d.names(route.For.Subject) || string(route.Operation.ID) != row.Operation {
			continue
		}
		for _, snapshot := range cut.Snapshots {
			if request, ok := match(snapshot, route, d.node, row); ok {
				found = append(found, request)
			}
		}
	}
	if len(found) != 1 {
		return driver.OperationRequest{}, fmt.Errorf("%w: %d matches", ErrNoRoute, len(found))
	}
	return found[0], nil
}

// names reports whether a route subject is the mounted resource. A mounted
// resource that carries no bridge id (the session names none) is matched by its
// resource id; a second route of the same resource id under another bridge is
// then a second match, which gives no write.
func (d *Dispatcher) names(subject driver.NodeRef) bool {
	if d.node.Bridge == "" {
		return subject.ID == d.node.ID
	}
	return subject == d.node
}

func match(snapshot semreg.Snapshot, route driver.SettingRoute, node driver.NodeRef, row composition.CommandRow) (driver.OperationRequest, bool) {
	r := route.Route
	var capability *semreg.CapabilityInstance
	for i := range snapshot.Capabilities {
		c := &snapshot.Capabilities[i]
		if c.InstanceID == r.CapabilityInstance && string(c.Definition.ID) == row.Capability {
			capability = c
		}
	}
	if capability == nil || capability.AssetID != snapshot.AssetID || capability.ServiceInstance != r.ServiceInstance ||
		capability.BindingID != r.BindingID || capability.SourceEpochID != r.SourceEpochID || capability.DriverGeneration != r.DriverGeneration {
		return driver.OperationRequest{}, false
	}
	var service *semreg.ServiceInstance
	for i := range snapshot.Services {
		s := &snapshot.Services[i]
		if s.InstanceID == r.ServiceInstance && s.Resource != nil && *s.Resource == node.ID {
			service = s
		}
	}
	if service == nil || service.AssetID != snapshot.AssetID || service.BindingID != r.BindingID ||
		service.SourceEpochID != r.SourceEpochID || service.DriverGeneration != r.DriverGeneration {
		return driver.OperationRequest{}, false
	}
	return driver.OperationRequest{
		AssetID:                            snapshot.AssetID,
		BindingID:                          r.BindingID,
		CapabilityInstanceID:               capability.InstanceID,
		Operation:                          route.Operation,
		ExpectedSemanticRevision:           snapshot.Revisions.Semantic,
		ExpectedCapabilityRevision:         snapshot.Revisions.Capabilities,
		ExpectedCapabilityInstanceRevision: capability.Revision,
		ExpectedDriverGeneration:           r.DriverGeneration,
		ExpectedSourceEpochID:              r.SourceEpochID,
		Preconditions:                      []operation.Precondition{},
	}, true
}
