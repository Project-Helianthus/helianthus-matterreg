package node

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	gfnode "thingfabric.dev/go-fabric/node"

	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Running is a started go-fabric node and the call that stops it.
type Running interface {
	Node() *gfnode.Node
	Stop(context.Context) error
}

// Launcher builds and starts a go-fabric node from a node configuration.
type Launcher interface {
	Launch(context.Context, gfnode.Config) (Running, error)
}

type sdkLauncher struct{}

type sdkRunning struct{ n *gfnode.Node }

func (sdkLauncher) Launch(ctx context.Context, cfg gfnode.Config) (Running, error) {
	n, err := gfnode.New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := n.Start(ctx); err != nil {
		_ = n.Stop(context.WithoutCancel(ctx))
		return nil, err
	}
	return sdkRunning{n}, nil
}

func (r sdkRunning) Node() *gfnode.Node { return r.n }

func (r sdkRunning) Stop(ctx context.Context) error { return r.n.Stop(ctx) }

// Class orders previously unseen resources: equipment and function resources
// come before component resources.
type Class int

// Resource classes.
const (
	ClassEquipmentFunction Class = iota
	ClassComponent
)

// Resource is one live SemReg resource handed to Apply: the catalog elements
// it carries, the relation keys it was composed from and the encoded value of
// each served attribute.
type Resource struct {
	ID           string
	Kind         string
	Class        Class
	Depth        int
	Elements     []string
	RelationKeys []string
	Values       map[Selector]Value
}

// PartReport is the report of one protocol part.
type PartReport struct {
	OwnerResourceID string
	Role            string
	Key             string
	DeviceType      uint32
	Rows            []string
}

// EndpointReport is the composition record of one endpoint.
type EndpointReport struct {
	ResourceIDs  []string
	RelationKeys []string
	Kind         string
	DeviceType   uint32
	Elements     []string
	Parts        []PartReport
}

// Refusal names a resource that has no endpoint and why. Missing lists the
// mandatory rows that have neither a transformed element nor a declared value.
type Refusal struct {
	ResourceID string
	Reason     string
	Missing    []string
}

// Report is the outcome of the last Apply.
type Report struct {
	Endpoints []EndpointReport
	Withheld  []publication.Withholding
	Refused   []Refusal
}

// Errors of the bridge lifecycle.
var (
	ErrStarted    = errors.New("node: bridge is already started")
	ErrNotStarted = errors.New("node: bridge is not started")
	ErrDuplicate  = errors.New("node: resource id is listed twice")
)

type entry struct {
	mount    *Mount
	rows     *partRows
	resource Resource
	shape    string
	applied  map[Selector]any
}

// Bridge is one bridge node over a store directory.
type Bridge struct {
	cfg      Config
	profiles map[string]composition.Profile
	rule     *publication.Rule
	launcher Launcher

	mu      sync.Mutex
	running Running
	node    *gfnode.Node
	mounted map[string]*entry
	report  Report

	cursor     driver.Cursor
	haveCursor bool

	operations driver.ScopedOperations
}

// SetOperations gives the bridge the scoped operations of the session whose
// SemanticInput the configuration carries. Endpoints composed afterwards
// dispatch their mapped writes and commands through them; without them a mapped
// write or command dispatches nothing and is refused.
func (b *Bridge) SetOperations(operations driver.ScopedOperations) {
	b.mu.Lock()
	b.operations = operations
	b.mu.Unlock()
}

// New validates cfg. It starts nothing.
func New(cfg Config) (*Bridge, error) { return NewWith(cfg, nil) }

// NewWith is New with the launcher that starts the SDK node; nil starts the
// SDK node directly.
func NewWith(cfg Config, launcher Launcher) (*Bridge, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	index, err := cfg.profileIndex()
	if err != nil {
		return nil, err
	}
	rule, err := publication.NewRule(cfg.Roster)
	if err != nil {
		return nil, err
	}
	if launcher == nil {
		launcher = sdkLauncher{}
	}
	return &Bridge{cfg: cfg, profiles: index, rule: rule, launcher: launcher, mounted: map[string]*entry{}}, nil
}

func (b *Bridge) nodeConfig() gfnode.Config {
	return gfnode.Config{
		Commissioning:      b.cfg.Commissioning,
		ProductDescription: b.cfg.ProductDescription,
		BasicInformation:   b.cfg.BasicInformation,
		Store:              gfnode.Store{Path: b.cfg.storePath()},
	}
}

// Start opens the store inside the store directory and starts the node. The
// attestation stays at the SDK's zero value, its test chain.
func (b *Bridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running != nil {
		return ErrStarted
	}
	if err := ensureDir(b.cfg.StoreDir); err != nil {
		return err
	}
	running, err := b.launcher.Launch(ctx, b.nodeConfig())
	if err != nil {
		return err
	}
	b.running, b.node = running, running.Node()
	b.mounted = map[string]*entry{}
	b.cursor, b.haveCursor = driver.Cursor{}, false
	return nil
}

// Stop stops the node. The store keeps the fabrics and the endpoint numbers.
func (b *Bridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running == nil {
		return nil
	}
	running := b.running
	b.running, b.node, b.mounted = nil, nil, map[string]*entry{}
	b.cursor, b.haveCursor = driver.Cursor{}, false
	return running.Stop(ctx)
}

// Node returns the running SDK node, nil before Start.
func (b *Bridge) Node() *gfnode.Node {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.node
}

// Numbers returns the endpoint number of every assembled endpoint by stable
// key, the roots and the protocol parts alike.
func (b *Bridge) Numbers() map[string]uint16 {
	b.mu.Lock()
	n := b.node
	b.mu.Unlock()
	out := map[string]uint16{}
	if n == nil {
		return out
	}
	if topology := n.Bridge().Topology(); topology != nil {
		for _, ep := range topology.Endpoints {
			if ep != nil && ep.SourceKey != nil {
				out[ep.SourceKey.String()] = ep.ID
			}
		}
	}
	return out
}

// Report returns the outcome of the last Apply.
func (b *Bridge) Report() Report {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.report
}

type plan struct {
	resource Resource
	rows     *partRows
	shape    string
	commands map[Selector][]composition.CommandRow
}

// Apply brings the endpoints to the given live resources. A resource that
// leaves is removed and its number stays reserved; a resource that returns
// gets its number back. A changed value of a served attribute is set on its
// server, which the SDK reports to subscribers; an unchanged value is not set.
func (b *Bridge) Apply(ctx context.Context, resources []Resource) (Report, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running == nil {
		return Report{}, ErrNotStarted
	}
	seen := map[string]struct{}{}
	live := []publication.Live{}
	for _, r := range resources {
		if _, ok := seen[r.ID]; ok {
			return Report{}, fmt.Errorf("%w: %q", ErrDuplicate, r.ID)
		}
		seen[r.ID] = struct{}{}
		for _, element := range r.Elements {
			live = append(live, publication.Live{ResourceID: r.ID, Element: element})
		}
	}
	decision := b.rule.Decide(live)
	report := Report{Withheld: decision.Withheld}
	admitted := map[string]map[string]struct{}{}
	for _, published := range decision.Published {
		set := map[string]struct{}{}
		for _, element := range published.Elements {
			set[element] = struct{}{}
		}
		admitted[published.ID] = set
	}
	plans := map[string]plan{}
	ordered := append([]Resource(nil), resources...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	for _, r := range ordered {
		set, ok := admitted[r.ID]
		if !ok {
			continue
		}
		p, refusal := b.plan(r, set)
		if refusal != nil {
			report.Refused = append(report.Refused, *refusal)
			continue
		}
		plans[r.ID] = p
	}
	agg := b.node.Aggregator()
	if err := b.reconcile(ctx, agg, plans, &report); err != nil {
		b.report = report
		return report, err
	}
	b.updateValues(plans, &report)
	b.finish(&report)
	b.report = report
	return report, nil
}

func (b *Bridge) plan(r Resource, admitted map[string]struct{}) (plan, *Refusal) {
	profile, ok := b.profiles[r.Kind]
	if !ok {
		return plan{}, &Refusal{ResourceID: r.ID, Reason: "no composition profile for kind " + r.Kind}
	}
	rows, err := classify(profile, admitted)
	if err != nil {
		return plan{}, &Refusal{ResourceID: r.ID, Reason: err.Error()}
	}
	if missing := missingOf(rows); len(missing) > 0 {
		return plan{}, &Refusal{ResourceID: r.ID, Reason: "mandatory rows have neither a transformed element nor a declared value", Missing: missing}
	}
	return plan{resource: r, rows: rows, shape: shapeOf(r.Kind, rows), commands: commandRows(profile)}, nil
}

func shapeOf(kind string, rows *partRows) string {
	var parts []string
	var walk func(*partRows)
	walk = func(p *partRows) {
		parts = append(parts, fmt.Sprintf("%s/%d", p.Role, p.DeviceTypeID))
		for _, d := range p.Dynamic {
			parts = append(parts, fmt.Sprintf("%s/%d/%d", d.PartRole, d.ClusterID, d.AttributeID))
		}
		for _, c := range p.Children {
			walk(c)
		}
	}
	walk(rows)
	return kind + "|" + strings.Join(parts, ",")
}

func (b *Bridge) reconcile(ctx context.Context, agg *gfnode.Aggregator, plans map[string]plan, report *Report) error {
	var removals, replacements []string
	for id, e := range b.mounted {
		p, ok := plans[id]
		switch {
		case !ok:
			removals = append(removals, id)
		case p.shape != e.shape:
			replacements = append(replacements, id)
		}
	}
	sort.Strings(removals)
	sort.Strings(replacements)
	var additions []plan
	for id, p := range plans {
		if _, ok := b.mounted[id]; !ok {
			additions = append(additions, p)
		}
	}
	sort.Slice(additions, func(i, j int) bool {
		a, c := additions[i].resource, additions[j].resource
		if a.Class != c.Class {
			return a.Class < c.Class
		}
		if a.Depth != c.Depth {
			return a.Depth < c.Depth
		}
		return a.ID < c.ID
	})
	built := map[string]*Mount{}
	ports := &CommandPorts{Input: b.cfg.Input, Operations: b.operations}
	compose := func(p plan) bool {
		mount, err := composeEndpoint(p.resource.ID, p.rows, p.resource.Values, p.commands, ports)
		if err != nil {
			report.Refused = append(report.Refused, Refusal{ResourceID: p.resource.ID, Reason: err.Error()})
			delete(plans, p.resource.ID)
			return false
		}
		built[p.resource.ID] = mount
		return true
	}
	var addList []plan
	for _, p := range additions {
		if compose(p) {
			addList = append(addList, p)
		}
	}
	var replaceList []plan
	for _, id := range replacements {
		if compose(plans[id]) {
			replaceList = append(replaceList, plans[id])
		} else {
			removals = append(removals, id)
		}
	}
	err := agg.Batch(ctx, func(batch *gfnode.Batch) error {
		for _, id := range removals {
			if err := batch.Remove(RootKey(id)); err != nil {
				return err
			}
		}
		for _, p := range replaceList {
			if err := batch.Replace(RootKey(p.resource.ID), built[p.resource.ID]); err != nil {
				return err
			}
		}
		for _, p := range addList {
			if err := batch.Add(built[p.resource.ID]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, id := range removals {
		delete(b.mounted, id)
	}
	for _, p := range append(replaceList, addList...) {
		b.mounted[p.resource.ID] = &entry{mount: built[p.resource.ID], rows: p.rows, resource: p.resource, shape: p.shape, applied: appliedOf(p.rows, p.resource.Values)}
	}
	return nil
}

func appliedOf(rows *partRows, values map[Selector]Value) map[Selector]any {
	out := map[Selector]any{}
	var walk func(*partRows)
	walk = func(p *partRows) {
		for _, d := range p.Dynamic {
			if v, ok := resolve(d.Row.MatterType, d.Row.Nullable, values, d.Selector); ok {
				out[d.Selector] = v
			}
		}
		for _, c := range p.Children {
			walk(c)
		}
	}
	walk(rows)
	return out
}

func resolve(matterType string, nullable bool, values map[Selector]Value, sel Selector) (any, bool) {
	value, ok := values[sel]
	if !ok {
		if nullable {
			return nil, true
		}
		return nil, false
	}
	v, err := stored(matterType, nullable, value)
	if err != nil {
		return nil, false
	}
	return v, true
}

func (b *Bridge) updateValues(plans map[string]plan, report *Report) {
	ids := make([]string, 0, len(b.mounted))
	for id := range b.mounted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := b.mounted[id]
		p, ok := plans[id]
		if !ok {
			continue
		}
		batches := map[Selector]map[uint32]any{}
		next := map[Selector]any{}
		var walk func(*partRows)
		walk = func(rows *partRows) {
			for _, d := range rows.Dynamic {
				v, ok := resolve(d.Row.MatterType, d.Row.Nullable, p.resource.Values, d.Selector)
				if !ok {
					continue
				}
				next[d.Selector] = v
				if old, had := e.applied[d.Selector]; had && old == v {
					continue
				}
				key := Selector{PartRole: d.PartRole, ClusterID: d.ClusterID}
				if batches[key] == nil {
					batches[key] = map[uint32]any{}
				}
				batches[key][d.AttributeID] = v
			}
			for _, c := range rows.Children {
				walk(c)
			}
		}
		walk(e.rows)
		for key, values := range batches {
			setter, ok := e.mount.Servers[key.PartRole][key.ClusterID]
			if !ok {
				continue
			}
			if err := setter.SetAttributes(values); err != nil {
				report.Refused = append(report.Refused, Refusal{ResourceID: id, Reason: fmt.Sprintf("value change refused: %v", err)})
				continue
			}
			for attr, v := range values {
				e.applied[Selector{key.PartRole, key.ClusterID, attr}] = v
			}
		}
		e.resource = p.resource
	}
}

func (b *Bridge) finish(report *Report) {
	ids := make([]string, 0, len(b.mounted))
	for id := range b.mounted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := b.mounted[id]
		report.Endpoints = append(report.Endpoints, endpointReport(e))
	}
	sort.Slice(report.Refused, func(i, j int) bool { return report.Refused[i].ResourceID < report.Refused[j].ResourceID })
}

func endpointReport(e *entry) EndpointReport {
	out := EndpointReport{
		ResourceIDs:  []string{e.resource.ID},
		RelationKeys: append([]string(nil), e.resource.RelationKeys...),
		Kind:         e.resource.Kind,
		DeviceType:   e.rows.DeviceTypeID,
		Elements:     append([]string(nil), e.resource.Elements...),
	}
	sort.Strings(out.RelationKeys)
	sort.Strings(out.Elements)
	var walk func(*partRows)
	walk = func(p *partRows) {
		for _, child := range p.Children {
			out.Parts = append(out.Parts, PartReport{
				OwnerResourceID: e.resource.ID, Role: child.Role, Key: PartKey(e.resource.ID, child.Role).String(),
				DeviceType: child.DeviceTypeID, Rows: append([]string(nil), child.Rows...),
			})
			walk(child)
		}
	}
	walk(e.rows)
	return out
}

func commandRows(profile composition.Profile) map[Selector][]composition.CommandRow {
	out := map[Selector][]composition.CommandRow{}
	for _, row := range profile.Commands {
		key := Selector{PartRole: row.PartKey, ClusterID: row.ClusterID, AttributeID: row.ElementID}
		out[key] = append(out[key], row)
	}
	return out
}
