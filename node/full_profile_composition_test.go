package node_test

import (
	"context"
	"slices"
	"sort"
	"strings"
	"testing"

	"thingfabric.dev/go-fabric/fabrictest"

	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/evse"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/solarpower"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	inverterKind    = "resource.inverter"
	liveElement     = "synthetic.inverter.live"
	aggregatorEP    = 1
	clusterElecPwr  = 0x0090
	clusterElecEnrg = 0x0091
	clusterPowerTop = 0x009C
	clusterPowerSrc = 0x002F
)

// shippedInverter is the Solar Power profile the repository ships, built from
// its profile package without any change.
func shippedInverter() composition.Profile {
	return composition.Profile{Kind: inverterKind, Attributes: solarpower.Rows(), Parts: solarpower.Parts()}
}

type managed struct {
	t        *testing.T
	site     *fabrictest.Site
	launcher *nodetest.SiteLauncher
	dir      string
	profiles []composition.Profile
	bridge   *node.Bridge
}

// newManaged runs the ordinary untagged node on the loopback interface with the
// given profiles and one admitted live element.
func newManaged(t *testing.T, profiles ...composition.Profile) *managed {
	t.Helper()
	site := fabrictest.NewSite(t, fabrictest.Options{Loopback: true})
	return &managed{t: t, site: site, launcher: &nodetest.SiteLauncher{Site: site}, dir: t.TempDir(), profiles: profiles}
}

func (m *managed) start(ctx context.Context) {
	m.t.Helper()
	cfg := nodetest.Config(m.dir)
	cfg.Profiles = m.profiles
	cfg.Roster = []publication.Row{{Element: liveElement, Kind: "attribute", Disposition: publication.Transformed}}
	bridge, err := node.NewWith(cfg, m.launcher)
	if err != nil {
		m.t.Fatal(err)
	}
	if err := bridge.Start(ctx); err != nil {
		m.t.Fatal(err)
	}
	m.bridge = bridge
}

func (m *managed) apply(ctx context.Context, id, kind string) node.Report {
	m.t.Helper()
	report, err := m.bridge.Apply(ctx, []node.Resource{{ID: id, Kind: kind, Elements: []string{liveElement}}})
	if err != nil {
		m.t.Fatalf("apply: %v", err)
	}
	return report
}

// expectedMissing is read from the shipped rows: every mandatory row with no
// source that the node reports by name (global attributes and Descriptor rows
// are supplied by the SDK).
func expectedMissing(p composition.Profile) []string {
	var out []string
	for _, part := range p.Parts {
		for _, r := range part.Attributes {
			global := r.AttributeID == 0xFFFD || (r.AttributeID >= 0xFFF8 && r.AttributeID < 0xFFFC)
			if r.Origin == composition.OriginMissing && r.Mandatory && r.ClusterID != 0x001D && !global {
				out = append(out, part.Role+":"+r.ReportName())
			}
		}
	}
	sort.Strings(out)
	return out
}

func sorted(in []string) []string {
	out := slices.Clone(in)
	sort.Strings(out)
	return out
}

// geometryProblems reports each mandatory nested device type the requirements
// tree names that the parts do not supply in its multiplicity, and each row
// that is not held by the part it belongs to.
func geometryProblems(requirements []partrows.Requirement, parts []composition.PartSpec) []string {
	var problems []string
	for _, req := range requirements {
		if req.Element != "device_type" || req.Conformance != "M" {
			continue
		}
		count := 0
		for _, part := range parts {
			if part.DeviceTypeID == req.ID {
				count += max(part.Count, 1)
			}
		}
		if count < max(req.Min, 1) {
			problems = append(problems, "missing part "+req.Name)
		}
	}
	for _, part := range parts {
		for _, r := range part.Attributes {
			if r.PartKey != part.Role {
				problems = append(problems, "row "+r.ReportName()+" of part "+r.PartKey+" sits in "+part.Role)
			}
		}
	}
	return problems
}

// synthetic returns a copy of the shipped profile in which the rows the target
// leaves without a source are given test-only declared values of a type the
// node converts, so that what the SDK still refuses is visible. The values are
// fixtures; they are not shipped and not a source.
func synthetic(p composition.Profile) composition.Profile {
	type fix struct {
		typ   string
		value any
	}
	table := map[[2]uint32]fix{
		{clusterPowerSrc, 0}: {"enum8", 1}, {clusterPowerSrc, 1}: {"uint8", 0}, {clusterPowerSrc, 5}: {"enum8", 0},
		{clusterElecPwr, 0}: {"enum8", 1}, {clusterElecPwr, 1}: {"uint8", 1}, {clusterElecPwr, 4}: {"int64", 230000},
		{clusterElecPwr, 5}: {"int64", 1000}, {clusterElecPwr, 8}: {"int64", 1000},
		{clusterElecPwr, 0xFFFC}: {"bitmap32", 2}, {clusterElecEnrg, 0xFFFC}: {"bitmap32", 6}, {clusterPowerTop, 0xFFFC}: {"bitmap32", 1},
	}
	out := p
	out.Parts = nil
	for _, part := range p.Parts {
		var rows []composition.AttributeRow
		for _, r := range part.Attributes {
			if r.Origin != composition.OriginMissing {
				rows = append(rows, r)
				continue
			}
			f, ok := table[[2]uint32{r.ClusterID, r.AttributeID}]
			if !ok || r.Field != "" {
				continue
			}
			r.MatterType, r.Origin, r.Declared = f.typ, composition.OriginDeclared, f.value
			rows = append(rows, r)
		}
		part.Attributes = rows
		out.Parts = append(out.Parts, part)
	}
	return out
}

// TestManagedFullProfileComposition runs the ordinary untagged node on the
// loopback interface with the Solar Power profile as shipped. The target maps
// no row for the inverter, so the positive composition of its two parts stays
// pending; what the node and the SDK do today is proven here.
func TestManagedFullProfileComposition(t *testing.T) {
	t.Run("the shipped inverter has no endpoint and no part over CASE, before and after a restart", func(t *testing.T) {
		m := newManaged(t, shippedInverter())
		ctx := nodetest.Bounded(t)
		m.start(ctx)
		want := expectedMissing(shippedInverter())
		if len(want) == 0 {
			t.Fatal("the shipped inverter has no missing mandatory row")
		}
		check := func(report node.Report) {
			t.Helper()
			if len(report.Endpoints) != 0 || len(m.bridge.Numbers()) != 0 {
				t.Fatalf("endpoints %+v numbers %v", report.Endpoints, m.bridge.Numbers())
			}
			if len(report.Refused) != 1 || report.Refused[0].ResourceID != "inv-a" || !slices.Equal(sorted(report.Refused[0].Missing), want) {
				t.Fatalf("refused %+v\nwant missing %v", report.Refused, want)
			}
		}
		check(m.apply(ctx, "inv-a", inverterKind))
		dev, err := m.launcher.Last()
		if err != nil {
			t.Fatal(err)
		}
		pair, err := m.site.AddController(fabrictest.ControllerOptions{}).Commission(ctx, dev, fabrictest.CommissionOptions{})
		if err != nil {
			t.Fatalf("commission: %v", err)
		}
		if !dev.Addr().IP.IsLoopback() {
			t.Fatalf("device address %v is not on the loopback interface", dev.Addr())
		}
		if parts := nodetest.Descriptor(ctx, t, m.site, pair, aggregatorEP).PartsList; len(parts) != 0 {
			t.Fatalf("Aggregator PartsList %v", parts)
		}
		if err := m.bridge.Stop(ctx); err != nil {
			t.Fatal(err)
		}
		m.start(ctx)
		check(m.apply(ctx, "inv-a", inverterKind))
		if parts := nodetest.Descriptor(ctx, t, m.site, pair, aggregatorEP).PartsList; len(parts) != 0 {
			t.Fatalf("Aggregator PartsList after restart %v", parts)
		}
	})

	t.Run("the shipped geometry holds both mandatory parts, each row at its own part", func(t *testing.T) {
		p := shippedInverter()
		if problems := geometryProblems(solarpower.Requirements(), p.Parts); len(problems) != 0 {
			t.Fatalf("shipped geometry %v", problems)
		}
		roles := map[string]uint32{}
		for _, part := range p.Parts {
			roles[part.Role] = part.DeviceTypeID
		}
		if len(roles) != 2 || roles[solarpower.RolePowerSource] != 0x0011 || roles[solarpower.RoleElectricalSensor] != 0x0510 {
			t.Fatalf("roles %v", roles)
		}
	})

	t.Run("an absent required part and a flat-only profile are not complete", func(t *testing.T) {
		p := shippedInverter()
		absent := slices.Clone(p.Parts[:1])
		if problems := geometryProblems(solarpower.Requirements(), absent); !slices.Equal(problems, []string{"missing part Electrical Sensor"}) {
			t.Fatalf("absent part %v", problems)
		}
		if problems := geometryProblems(solarpower.Requirements(), nil); len(problems) != 2 {
			t.Fatalf("flat-only %v", problems)
		}
	})

	t.Run("a part row at the wrong location is refused by the SDK on the real node", func(t *testing.T) {
		good := synthetic(shippedInverter())
		good.Kind = "resource.synthetic.located"
		moved := synthetic(shippedInverter())
		moved.Kind = "resource.synthetic.moved"
		moved.Parts = slices.Clone(moved.Parts)
		moved.Parts[0].Attributes = append(slices.Clone(moved.Parts[0].Attributes), moved.Parts[1].Attributes...)
		moved.Parts[1].Attributes = nil
		if problems := geometryProblems(solarpower.Requirements(), moved.Parts); len(problems) == 0 {
			t.Fatal("the geometry check missed the moved rows")
		}
		m := newManaged(t, good, moved)
		ctx := nodetest.Bounded(t)
		m.start(ctx)
		report, err := m.bridge.Apply(ctx, []node.Resource{
			{ID: "inv-good", Kind: good.Kind, Elements: []string{liveElement}},
			{ID: "inv-moved", Kind: moved.Kind, Elements: []string{liveElement}},
		})
		if err != nil {
			t.Fatal(err)
		}
		reasons := map[string]string{}
		for _, r := range report.Refused {
			reasons[r.ResourceID] = r.Reason
		}
		if !strings.Contains(reasons["inv-moved"], "PowerSource has no server cluster requirement for cluster 0x0090") {
			t.Fatalf("moved reasons %v", reasons)
		}
		if reason := reasons["inv-good"]; !strings.Contains(reason, "Accuracy") || strings.Contains(reason, "no server cluster requirement") {
			t.Fatalf("with the rows in place the SDK still lacks only Accuracy; got %q", reason)
		}
	})

	t.Run("no writable row exists in the shipped composition", func(t *testing.T) {
		p := shippedInverter()
		if len(p.Commands) != 0 {
			t.Fatalf("commands %v", p.Commands)
		}
		fed := 0
		for _, set := range [][]composition.AttributeRow{evse.Rows(), solarpower.Rows()} {
			for _, r := range set {
				if r.Origin == composition.OriginSemreg {
					fed++
				}
			}
		}
		for _, part := range append(evse.Parts(), solarpower.Parts()...) {
			for _, r := range part.Attributes {
				if r.Origin == composition.OriginSemreg {
					fed++
					if r.Name != "ActiveCurrent" || r.Element != "evse.ac.current" {
						t.Fatalf("unexpected mapped row %s", r.ReportName())
					}
				}
			}
		}
		if fed != 1 {
			t.Fatalf("%d mapped rows; the target maps one attribute and no operation", fed)
		}
		t.Log("pending: the CASE write or invoke that reaches a SemReg owner needs a mapped operation; the target maps none")
	})
}
