// Package driver is the public SemReg driver factory of the Matter bridge. A
// host that links it calls NewFactory, binds the factory's two options on its
// flag set (or passes the same options directly) and hands the factory to
// SemReg. The factory delegates the node to package node and imports neither
// the SemReg host nor its runtime. It builds without any build tag.
package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"sync"

	semdriver "github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	gfnode "thingfabric.dev/go-fabric/node"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

const (
	// Kind is the driver kind of the bridge.
	Kind semdriver.DriverKind = "matter"
	// FlagEnabled is the flag that enables the library.
	FlagEnabled = "matter-library-enabled"
	// FlagEndpointStore is the flag that names the store directory.
	FlagEndpointStore = "matter-endpoint-store"
	// DefaultInstance is the instance key BindFlags gives its options.
	DefaultInstance semdriver.InstanceKey = "matter"

	storeProvider     = "matter.endpoint-store"
	storeAccessDomain = "filesystem"

	developmentPasscode      uint32 = 20202021
	developmentDiscriminator uint16 = 3840
	developmentIterations    uint32 = 1000
)

// Errors of planning and construction.
var (
	ErrOptions = errors.New("driver: invalid options")
	ErrPlan    = errors.New("driver: plan does not describe one bridge instance")
)

// Options is the content of Config.Options.
type Options struct {
	// EndpointStore is the directory that holds the bridge's durable store.
	EndpointStore string `json:"endpoint_store"`
}

type factory struct {
	profiles []composition.Profile
	roster   []publication.Row
	loadErr  error
}

var _ semdriver.Factory = (*factory)(nil)

// NewFactory returns the factory. It loads the embedded composition profiles
// once, here.
func NewFactory() *factory {
	f := &factory{}
	f.profiles, f.roster, f.loadErr = load()
	return f
}

func load() ([]composition.Profile, []publication.Row, error) {
	source, err := composition.Load()
	if err != nil {
		return nil, nil, err
	}
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		return nil, nil, fmt.Errorf("driver: no composition profile for %s", pilot.ResourceKind)
	}
	return []composition.Profile{profile}, rosterOf([]composition.Profile{profile}), nil
}

func rosterOf(profiles []composition.Profile) []publication.Row {
	seen := map[string]struct{}{}
	var add func(rows []composition.AttributeRow)
	add = func(rows []composition.AttributeRow) {
		for _, row := range rows {
			if row.Origin == composition.OriginSemreg && row.Element != "" {
				seen[row.Element] = struct{}{}
			}
		}
	}
	var walk func(parts []composition.PartSpec)
	walk = func(parts []composition.PartSpec) {
		for _, part := range parts {
			add(part.Attributes)
			walk(part.Children)
		}
	}
	for _, profile := range profiles {
		add(profile.Attributes)
		walk(profile.Parts)
	}
	elements := make([]string, 0, len(seen))
	for element := range seen {
		elements = append(elements, element)
	}
	sort.Strings(elements)
	rows := make([]publication.Row, len(elements))
	for i, element := range elements {
		rows[i] = publication.Row{Element: element, Kind: "attribute", Disposition: publication.Transformed}
	}
	return rows
}

// BindFlags installs the factory's two options on fs. The returned closure,
// read after fs is parsed, yields the enabled state and the opaque options for
// Plan. No flag state is kept outside fs.
func (f *factory) BindFlags(fs *flag.FlagSet) func() (enabled bool, options semdriver.Config) {
	enabled := fs.Bool(FlagEnabled, false, "enable the Matter library bridge")
	store := fs.String(FlagEndpointStore, "", "directory of the Matter bridge's durable endpoint and fabric store")
	return func() (bool, semdriver.Config) {
		raw, _ := json.Marshal(Options{EndpointStore: *store})
		return *enabled, semdriver.Config{Instance: string(DefaultInstance), Enabled: *enabled, Options: raw}
	}
}

// Descriptor declares no family: the bridge is northbound and publishes nothing
// into SemReg.
func (f *factory) Descriptor() semdriver.Descriptor { return semdriver.Descriptor{} }

// Plan is pure. A disabled instance plans nothing; an enabled one plans one
// instance whose single transport claim names the store directory.
func (f *factory) Plan(cfg semdriver.Config) ([]semdriver.InstancePlan, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	options, err := decodeOptions(cfg.Options)
	if err != nil {
		return nil, err
	}
	plan := semdriver.InstancePlan{
		Driver:   Kind,
		Instance: semdriver.InstanceKey(cfg.Instance),
		Transports: []semdriver.TransportClaim{{
			Provider: storeProvider, ConfigKey: options.EndpointStore,
			AccessDomain: storeAccessDomain, Share: semdriver.ShareExclusive,
		}},
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	return []semdriver.InstancePlan{plan}, nil
}

func decodeOptions(raw json.RawMessage) (Options, error) {
	var options Options
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&options); err != nil {
		return Options{}, fmt.Errorf("%w: %v", ErrOptions, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Options{}, fmt.Errorf("%w: trailing content", ErrOptions)
	}
	if options.EndpointStore == "" {
		return Options{}, fmt.Errorf("%w: endpoint_store is empty", ErrOptions)
	}
	return options, nil
}

// Construct builds the instance of one planned bridge. It starts nothing.
func (f *factory) Construct(_ context.Context, plan semdriver.InstancePlan) (semdriver.Instance, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	if plan.Driver != Kind || len(plan.Transports) != 1 || plan.Transports[0].Provider != storeProvider {
		return nil, ErrPlan
	}
	cfg := node.Config{
		StoreDir: plan.Transports[0].ConfigKey,
		Profiles: f.profiles,
		Roster:   f.roster,
		ProductDescription: gfnode.ProductDescription{
			VendorID: node.DevelopmentVendorID, ProductID: node.ProductID,
		},
		Commissioning: gfnode.Commissioning{
			Passcode: developmentPasscode, Discriminator: developmentDiscriminator,
			PBKDFIterations: developmentIterations, Salt: []byte("SPAKE2P Key Salt"),
		},
		BasicInformation: gfnode.BasicInformation{
			VendorName: "Helianthus", ProductName: "Helianthus Matter bridge", NodeLabel: "Helianthus",
			SerialNumber: "development", HardwareVersion: 1, HardwareVersionString: "1",
			SoftwareVersion: 1, SoftwareVersionString: "1",
		},
	}
	if _, err := node.New(cfg); err != nil {
		return nil, err
	}
	return &instance{cfg: cfg}, nil
}

type instance struct {
	cfg node.Config

	mu     sync.Mutex
	bridge *node.Bridge
}

// Start builds the bridge over the session's SemanticInput, starts it and reads
// SemReg once.
func (i *instance) Start(ctx context.Context, session semdriver.Session) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.bridge != nil {
		return node.ErrStarted
	}
	cfg := i.cfg
	cfg.Input = session.SemanticInput()
	bridge, err := node.New(cfg)
	if err != nil {
		return err
	}
	if err := bridge.Start(ctx); err != nil {
		return err
	}
	if _, err := bridge.Refresh(ctx); err != nil {
		_ = bridge.Stop(context.WithoutCancel(ctx))
		return err
	}
	i.bridge = bridge
	return nil
}

// FenceWithdrawal is nonblocking and calls nothing.
func (i *instance) FenceWithdrawal() {}

// Stop stops the bridge.
func (i *instance) Stop(ctx context.Context) semdriver.CloseReport {
	i.mu.Lock()
	bridge := i.bridge
	i.bridge = nil
	i.mu.Unlock()
	if bridge == nil {
		return semdriver.CloseReport{Confirmed: true}
	}
	if err := bridge.Stop(ctx); err != nil {
		return semdriver.CloseReport{Err: err}
	}
	return semdriver.CloseReport{Confirmed: true}
}
