package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"thingfabric.dev/go-fabric/cluster/spec"
	"thingfabric.dev/go-fabric/contract"

	"github.com/Project-Helianthus/helianthus-matterreg/command"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// CommandPorts are the session's ports the command dispatch acts through.
type CommandPorts struct {
	Bridge     driver.BridgeID
	Input      driver.SemanticInput
	Operations driver.ScopedOperations
}

// Command errors; neither carries an Interaction Model status.
var (
	ErrUnmapped  = errors.New("node: the attribute has no command row")
	ErrNoVeto    = errors.New("node: the cluster server takes no write veto")
	errNoHandler = errors.New("node: the cluster server takes no command handlers")
)

// mountCommands installs on one cluster server the one write veto and a handler
// for each request command that has a row. rows are the rows of that part and
// cluster. Without ports the veto and the handlers dispatch nothing.
func mountCommands(ref driver.NodeRef, srv contract.ClusterServer, rows []composition.CommandRow, ports *CommandPorts) error {
	var dispatcher *command.Dispatcher
	if ports != nil {
		dispatcher = command.New(ref, ports.Input, ports.Operations)
	}
	byID := make(map[uint32]composition.CommandRow, len(rows))
	for _, row := range rows {
		byID[row.ElementID] = row
	}
	veto, ok := srv.(spec.BeforeWriteSetter)
	if !ok {
		return fmt.Errorf("%w: %T", ErrNoVeto, srv)
	}
	veto.SetBeforeWrite(func(ctx context.Context, attrID uint32, value any) error {
		row, ok := byID[attrID]
		if !ok {
			return ErrUnmapped
		}
		return dispatcher.Write(ctx, row, value)
	})
	if len(rows) == 0 {
		return nil
	}
	registrar, ok := srv.(spec.CommandRegistrar)
	if !ok {
		return fmt.Errorf("%w: %T", errNoHandler, srv)
	}
	for _, row := range rows {
		err := registrar.Handle(row.ElementID, func(ctx context.Context, fields any) (any, error) {
			return nil, dispatcher.Invoke(ctx, row, fields)
		})
		if err != nil && !errors.Is(err, spec.ErrNotAccepted) {
			return err
		}
	}
	return nil
}

// rowsOf returns the rows of one part and cluster.
func rowsOf(commands map[Selector][]composition.CommandRow, role string, cluster uint32) []composition.CommandRow {
	var out []composition.CommandRow
	for selector, rows := range commands {
		if selector.PartRole == role && selector.ClusterID == cluster {
			out = append(out, rows...)
		}
	}
	return out
}
