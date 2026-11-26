package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/graph"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// ErrNoInput is returned by Refresh when the configuration carries no
// SemanticInput.
var ErrNoInput = errors.New("node: no SemanticInput is configured")

const (
	encoderAmpereToMilliampere = "ampere_to_milliampere"
	encoderKelvinToCentiCelsus = "kelvin_to_centicelsius"
)

// Refresh reads SemReg once through the configured SemanticInput and brings
// the endpoints to what it read. The first Refresh after Start, and a Refresh
// answered with a resync, reads a new Capture and its lens; any other Refresh
// reads the lens of each commit Resume returns, in order. Nothing is written to
// SemReg.
func (b *Bridge) Refresh(ctx context.Context) (Report, error) {
	input := b.cfg.Input
	if input == nil {
		return Report{}, ErrNoInput
	}
	b.mu.Lock()
	cursor, have := b.cursor, b.haveCursor
	b.mu.Unlock()
	changes, err := input.Resume(ctx, cursor, driver.Filter{})
	if err != nil {
		return Report{}, err
	}
	if err := changes.Validate(); err != nil {
		return Report{}, err
	}
	var last Report
	if changes.Resync || !have {
		cut, err := input.Capture(ctx, driver.Filter{})
		if err != nil {
			return Report{}, err
		}
		if last, err = b.applyCut(ctx, input, cut); err != nil {
			return last, err
		}
	} else {
		for _, commit := range changes.Commits {
			if last, err = b.applyCut(ctx, input, commit); err != nil {
				return last, err
			}
		}
		if len(changes.Commits) == 0 {
			last = b.Report()
		}
	}
	b.mu.Lock()
	b.cursor, b.haveCursor = changes.Cursor, true
	b.mu.Unlock()
	return last, nil
}

func (b *Bridge) applyCut(ctx context.Context, input driver.SemanticInput, cut driver.SemanticCut) (Report, error) {
	if err := cut.Validate(); err != nil {
		return Report{}, err
	}
	lens, err := readLens(ctx, input, cut.Ref, b.cfg.Lens)
	if err != nil {
		return Report{}, err
	}
	return b.Apply(ctx, b.resourcesOf(lens))
}

func readLens(ctx context.Context, input driver.SemanticInput, ref graph.CutRef, request graph.LensRequest) (graph.LensResult, error) {
	var all graph.LensResult
	for page := 0; ; page++ {
		result, err := input.RetrieveLens(ctx, ref, request)
		if err != nil {
			return graph.LensResult{}, err
		}
		if page == 0 {
			all = result
		} else {
			all.Resources = append(all.Resources, result.Resources...)
			all.Relations = append(all.Relations, result.Relations...)
			all.Facts = append(all.Facts, result.Facts...)
		}
		if result.NextPage == nil || *result.NextPage == "" {
			all.NextPage = nil
			return all, nil
		}
		request.PageToken = *result.NextPage
	}
}

// resourcesOf turns a lens result into the live resources of Apply: for each
// reached resource its kind, class, depth, the fact elements it carries, the
// relation keys it appears in and the encoded value of each served attribute
// that has a usable current value.
func (b *Bridge) resourcesOf(lens graph.LensResult) []Resource {
	factsBy := map[string][]graph.LensFact{}
	for _, fact := range lens.Facts {
		factsBy[string(fact.Subject)] = append(factsBy[string(fact.Subject)], fact)
	}
	relationsBy := map[string][]string{}
	for _, relation := range lens.Relations {
		if !relation.InPlane {
			continue
		}
		key := RelationKey(string(relation.Definition.ID), string(relation.From), string(relation.To))
		relationsBy[string(relation.From)] = append(relationsBy[string(relation.From)], key)
		if relation.To != relation.From {
			relationsBy[string(relation.To)] = append(relationsBy[string(relation.To)], key)
		}
	}
	out := make([]Resource, 0, len(lens.Resources))
	for _, reached := range lens.Resources {
		id := string(reached.Resource.ResourceID)
		resource := Resource{
			ID: id, Kind: string(reached.Resource.Kind), Depth: reached.Depth,
			RelationKeys: relationsBy[id], Values: map[Selector]Value{},
		}
		if reached.Resource.Anchor.Class == semreg.AnchorComponent {
			resource.Class = ClassComponent
		}
		seen := map[string]struct{}{}
		for _, fact := range factsBy[id] {
			element := string(fact.Key.FactID)
			if _, ok := seen[element]; !ok {
				seen[element] = struct{}{}
				resource.Elements = append(resource.Elements, element)
			}
		}
		sort.Strings(resource.Elements)
		if profile, ok := b.profiles[resource.Kind]; ok {
			collectValues(&resource, profile, factsBy[id])
		}
		out = append(out, resource)
	}
	return out
}

// RelationKey is the key of one relation instance in a target report: its
// definition, from and to.
func RelationKey(definition, from, to string) string {
	return definition + ":" + from + ">" + to
}

func collectValues(resource *Resource, profile composition.Profile, facts []graph.LensFact) {
	rows := func(role string, attributes []composition.AttributeRow) {
		for _, row := range attributes {
			if row.Origin != composition.OriginSemreg || row.Field != "" {
				continue
			}
			value, ok := currentValue(row, facts)
			if !ok {
				continue
			}
			encoded, err := encode(row.Encoder, value)
			if err != nil {
				continue
			}
			resource.Values[Selector{PartRole: role, ClusterID: row.ClusterID, AttributeID: row.AttributeID}] = Value{Value: encoded}
		}
	}
	rows("", profile.Attributes)
	var walk func([]composition.PartSpec)
	walk = func(parts []composition.PartSpec) {
		for _, part := range parts {
			rows(part.Role, part.Attributes)
			walk(part.Children)
		}
	}
	walk(profile.Parts)
}

// currentValue is the one usable value of the row's element on the resource:
// an observed, qualified, promoted, good, available and fresh candidate that
// the evaluation also reports fresh and available. No usable value, a conflict
// or more than one candidate is unknown.
func currentValue(row composition.AttributeRow, facts []graph.LensFact) (semreg.Value, bool) {
	var found []semreg.Value
	for _, fact := range facts {
		if !rowKey(row, fact.Key) {
			continue
		}
		if len(fact.Conflicts) != 0 {
			return semreg.Value{}, false
		}
		for _, candidate := range fact.Candidates {
			if candidate.Value != nil && usable(candidate) && evaluated(candidate.CandidateID, fact.Evaluation) {
				found = append(found, *candidate.Value)
			}
		}
	}
	if len(found) != 1 {
		return semreg.Value{}, false
	}
	return found[0], true
}

func rowKey(row composition.AttributeRow, key semreg.FactKey) bool {
	if string(key.FactID) != row.Element {
		return false
	}
	if row.PackID != "" && (string(key.PackID) != row.PackID || string(key.PackVersion) != row.PackVersion) {
		return false
	}
	if row.Dimension == "" {
		return true
	}
	for _, dimension := range key.Dimensions {
		if string(dimension.ID) == row.Dimension {
			return true
		}
	}
	return false
}

func usable(candidate semreg.FactCandidate) bool {
	q := candidate.Quality
	return q.Assertion == semreg.AssertionObserved && q.Qualification == semreg.QualificationQualified &&
		q.Promotion == semreg.PromotionPromoted && q.Validity == semreg.ValidityGood &&
		q.Availability == semreg.AvailabilityAvailable && q.Freshness == semreg.FreshnessFresh
}

func evaluated(id semreg.CandidateID, items []semreg.EvaluatedFact) bool {
	for _, item := range items {
		if item.CandidateID == id {
			return item.Freshness == semreg.FreshnessFresh && item.EffectiveAvailability == semreg.AvailabilityAvailable
		}
	}
	return false
}

// encode converts the SI value of a SemReg fact to the integer a Matter
// attribute carries. A value the encoding cannot represent exactly is an error.
func encode(encoder string, value semreg.Value) (int64, error) {
	switch encoder {
	case encoderAmpereToMilliampere:
		return scaled(value, "unit.ampere", big.NewRat(1000, 1), new(big.Rat))
	case encoderKelvinToCentiCelsus:
		return scaled(value, "unit.kelvin", big.NewRat(100, 1), big.NewRat(-27315, 1))
	}
	return 0, fmt.Errorf("encoder %q is not known", encoder)
}

func scaled(value semreg.Value, unit semreg.DefinitionID, factor, offset *big.Rat) (int64, error) {
	if value.Kind != semreg.ValueQuantity || value.Quantity == nil || value.Quantity.Unit != unit {
		return 0, fmt.Errorf("value is not a quantity in %s", unit)
	}
	coefficient, ok := new(big.Int).SetString(value.Quantity.Number.Coefficient, 10)
	if !ok {
		return 0, fmt.Errorf("coefficient is not an integer")
	}
	exponent := int64(value.Quantity.Number.Exponent10)
	if exponent > 64 || exponent < -64 {
		return 0, fmt.Errorf("exponent is out of range")
	}
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(max(exponent, -exponent)), nil)
	number := new(big.Rat).SetInt(coefficient)
	if exponent >= 0 {
		number.Mul(number, new(big.Rat).SetInt(power))
	} else {
		number.Quo(number, new(big.Rat).SetInt(power))
	}
	number.Mul(number, factor)
	number.Add(number, offset)
	if !number.IsInt() || !number.Num().IsInt64() {
		return 0, fmt.Errorf("value is not an exact integer of the Matter unit")
	}
	return number.Num().Int64(), nil
}
