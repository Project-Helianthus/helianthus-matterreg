package nodetest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/graph"
)

// Fact is one fact of a synthetic resource: the element it states and its
// value; a nil Value is a candidate-less, unknown fact.
type Fact struct {
	Element string
	Value   *semreg.Value
	// Dimension, when set, is a dimension id the fact key carries.
	Dimension string
}

// Item is one resource of a synthetic lens.
type Item struct {
	ID        string
	Kind      string
	Component bool
	Depth     int
	Facts     []Fact
}

// Relation is one in-plane relation instance of a synthetic lens.
type Relation struct{ Definition, From, To string }

// Kelvin is a quantity in kelvin. The coefficient is decimal text.
func Kelvin(coefficient string, exponent10 int32) *semreg.Value {
	return &semreg.Value{Kind: semreg.ValueQuantity, Quantity: &semreg.Quantity{
		Number: semreg.Decimal{Coefficient: coefficient, Exponent10: exponent10}, Unit: "unit.kelvin",
	}}
}

type commit struct {
	position uint64
	ref      graph.CutRef
}

// Input is a synthetic driver.SemanticInput that holds lens results by cut and
// the commits made after each position. It follows the host's cursor rules: the
// zero cursor stands for the position of the session's latest capture, and is
// an error before any capture; a cursor of another incarnation is answered with
// a resync that echoes the cursor it was given. It counts every call.
type Input struct {
	mu          sync.Mutex
	lenses      map[semreg.Digest]graph.LensResult
	current     semreg.Digest
	position    uint64
	incarnation string
	commits     []commit
	captured    bool
	capturedAt  uint64
	Captures    int
	Resumes     int
	Retrieved   int
}

// NewInput returns an input that holds no cut.
func NewInput() *Input {
	return &Input{lenses: map[semreg.Digest]graph.LensResult{}, incarnation: "incarnation.synthetic"}
}

// Reincarnate makes every cursor handed out so far stale, as a host restart does.
func (i *Input) Reincarnate() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.incarnation += "+"
}

// Publish makes the lens the current cut and records one commit that carries it.
func (i *Input) Publish(items []Item, relations []Relation) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.position++
	sum := sha256.Sum256([]byte(fmt.Sprintf("cut-%d", i.position)))
	id := semreg.Digest("sha256:" + hex.EncodeToString(sum[:]))
	result := graph.LensResult{Cut: graph.CutRef{CutID: id, Completeness: graph.CutComplete}}
	for _, item := range items {
		anchor := semreg.ResourceAnchor{Class: semreg.AnchorEquipment}
		if item.Component {
			anchor.Class = semreg.AnchorComponent
		}
		result.Resources = append(result.Resources, graph.LensResource{
			Resource: semreg.Resource{ResourceID: semreg.ResourceID(item.ID), Kind: semreg.DefinitionID(item.Kind), Anchor: anchor},
			Depth:    item.Depth,
		})
		for _, fact := range item.Facts {
			lf := graph.LensFact{Subject: semreg.ResourceID(item.ID), Key: semreg.FactKey{FactID: semreg.DefinitionID(fact.Element)}}
			if fact.Dimension != "" {
				lf.Key.Dimensions = []semreg.Dimension{{ID: semreg.DefinitionID(fact.Dimension)}}
			}
			if fact.Value != nil {
				candidate := semreg.CandidateID(fmt.Sprintf("%s/%s", item.ID, fact.Element))
				lf.Candidates = []semreg.FactCandidate{{
					CandidateID: candidate, Value: fact.Value,
					Quality: semreg.Quality{
						Assertion: semreg.AssertionObserved, Qualification: semreg.QualificationQualified, Promotion: semreg.PromotionPromoted,
						Validity: semreg.ValidityGood, Availability: semreg.AvailabilityAvailable, Freshness: semreg.FreshnessFresh,
					},
				}}
				lf.Evaluation = []semreg.EvaluatedFact{{CandidateID: candidate, Freshness: semreg.FreshnessFresh, EffectiveAvailability: semreg.AvailabilityAvailable}}
			}
			result.Facts = append(result.Facts, lf)
		}
	}
	for _, r := range relations {
		result.Relations = append(result.Relations, graph.LensRelation{
			Definition: semreg.DefinitionRef{ID: semreg.DefinitionID(r.Definition)},
			From:       semreg.ResourceID(r.From), To: semreg.ResourceID(r.To), InPlane: true,
		})
	}
	i.lenses[id] = result
	i.current = id
	i.commits = append(i.commits, commit{position: i.position, ref: result.Cut})
}

func (i *Input) cursor() driver.Cursor {
	return driver.Cursor{Incarnation: i.incarnation, Position: semreg.Uint64(fmt.Sprint(i.position))}
}

// Capture implements driver.SemanticInput. It records the position the zero
// cursor then stands for.
func (i *Input) Capture(context.Context, driver.Filter) (driver.SemanticCut, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Captures++
	i.captured, i.capturedAt = true, i.position
	return driver.SemanticCut{Ref: i.lenses[i.current].Cut}, nil
}

// Resume implements driver.SemanticInput. The zero cursor stands for the
// position of the latest capture and is an error when nothing was captured; a
// cursor of another incarnation is a resync that echoes that cursor; otherwise
// the commits after the cursor are returned with a cursor at the present.
func (i *Input) Resume(_ context.Context, cursor driver.Cursor, _ driver.Filter) (driver.Changes, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Resumes++
	from := i.capturedAt
	if cursor == (driver.Cursor{}) {
		if !i.captured {
			return driver.Changes{}, errors.New("nodetest: no capture to resume from")
		}
	} else {
		if err := cursor.Validate(); err != nil {
			return driver.Changes{}, err
		}
		if cursor.Incarnation != i.incarnation {
			return driver.Changes{Cursor: cursor, Resync: true}, nil
		}
		n, err := strconv.ParseUint(string(cursor.Position), 10, 64)
		if err != nil {
			return driver.Changes{}, err
		}
		from = n
	}
	out := driver.Changes{Cursor: i.cursor()}
	for _, c := range i.commits {
		if c.position > from {
			out.Commits = append(out.Commits, driver.SemanticCut{Ref: c.ref})
		}
	}
	return out, nil
}

// RetrieveLens implements driver.SemanticInput.
func (i *Input) RetrieveLens(_ context.Context, ref graph.CutRef, _ graph.LensRequest) (graph.LensResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Retrieved++
	result, ok := i.lenses[ref.CutID]
	if !ok {
		return graph.LensResult{}, errors.New("nodetest: unknown cut")
	}
	return result, nil
}

// SumMembers implements driver.SemanticInput; the bridge does not call it.
func (i *Input) SumMembers(context.Context, graph.CutRef, graph.LensRequest, semreg.DefinitionRef, semreg.ResourceID) (graph.MemberSum, error) {
	return graph.MemberSum{}, errors.New("nodetest: SumMembers is not used")
}
