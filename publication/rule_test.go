package publication

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// Synthetic catalog elements. The names follow device families and are not
// rows of any real roster.
const (
	current     = "synthetic.heat-pump-family.electrical-current"
	power       = "synthetic.heat-pump-family.electrical-power"
	unknown     = "synthetic.heat-generator-family.operating-status"
	excluded    = "synthetic.ventilation-family.fan-speed"
	unsupported = "synthetic.heat-generator-family.service"
	withheld    = "synthetic.heat-generator-family.contains"
	blank       = "synthetic.heat-generator-family.no-disposition"
	relation    = "synthetic.heat-pump-family.hosts"
	unmapped    = "synthetic.heat-pump-family.unmapped"

	resA = "synthetic-resource-a"
	resB = "synthetic-resource-b"
	resC = "synthetic-resource-c"
	resD = "synthetic-resource-d"
)

func syntheticRows() []Row {
	return []Row{
		{Element: current, Kind: "fact", Disposition: Transformed},
		{Element: power, Kind: "fact", Disposition: Transformed},
		{Element: unknown, Kind: "fact", Disposition: Unknown},
		{Element: excluded, Kind: "capability", Disposition: Transformed},
		{Element: unsupported, Kind: "service", Disposition: Unsupported},
		{Element: withheld, Kind: "relation", Disposition: Withheld},
		{Element: blank, Kind: "fact", Disposition: ""},
		{Element: relation, Kind: KindRelation, Disposition: Transformed},
	}
}

func exclusionDocument(elements ...string) []byte {
	if elements == nil {
		elements = []string{}
	}
	data, err := json.Marshal(map[string][]string{"elements": elements})
	if err != nil {
		panic(err)
	}
	return data
}

func mustRule(t *testing.T, rows []Row, exclusions []byte) *Rule {
	t.Helper()
	rule, err := newRule(rows, exclusions)
	if err != nil {
		t.Fatalf("newRule: %v", err)
	}
	return rule
}

type recorder struct {
	saved [][]byte
	fail  error
}

func (r *recorder) save(document []byte) error {
	if r.fail != nil {
		return r.fail
	}
	r.saved = append(r.saved, append([]byte(nil), document...))
	return nil
}

func (r *recorder) last(t *testing.T) []byte {
	t.Helper()
	if len(r.saved) == 0 {
		t.Fatal("no document was saved")
	}
	return r.saved[len(r.saved)-1]
}

func refuseSave(t *testing.T) SaveFunc {
	return func([]byte) error {
		t.Helper()
		t.Fatal("a store that assigns nothing new must not save")
		return nil
	}
}

func syntheticID(index int) string { return fmt.Sprintf("synthetic-resource-%03d", index) }

// The rule publishes two resources that share one transformed element and
// withholds an unknown element, an excluded element and a resource with an
// empty id. The published resources survive a save and a second load.
func TestPublishedResourcesKeepTheirNumbersAcrossALoad(t *testing.T) {
	rule := mustRule(t, syntheticRows(), exclusionDocument(excluded))
	decision := rule.Decide([]Live{
		{ResourceID: resA, Element: current},
		{ResourceID: resB, Element: current},
		{ResourceID: resC, Element: unknown},
		{ResourceID: resD, Element: excluded},
		{ResourceID: "", Element: current},
	})
	wantPublished := []Resource{
		{ID: resA, Elements: []string{current}},
		{ID: resB, Elements: []string{current}},
	}
	if !reflect.DeepEqual(decision.Published, wantPublished) {
		t.Fatalf("published = %#v, want %#v", decision.Published, wantPublished)
	}
	wantWithheld := []Withholding{
		{ResourceID: "", Element: current, Reason: ReasonResourceID},
		{ResourceID: resC, Element: unknown, Reason: "element disposition is unknown"},
		{ResourceID: resD, Element: excluded, Reason: ReasonExcluded},
	}
	if !reflect.DeepEqual(decision.Withheld, wantWithheld) {
		t.Fatalf("withheld = %#v, want %#v", decision.Withheld, wantWithheld)
	}

	var disk recorder
	store := NewStore()
	first := store.Assign(decision.ResourceIDs(), disk.save)
	if want := map[string]uint16{resA: 1, resB: 2}; !reflect.DeepEqual(first.Endpoints, want) || len(first.Refused) != 0 {
		t.Fatalf("first assignment = %#v", first)
	}

	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second := reloaded.Assign(decision.ResourceIDs(), refuseSave(t))
	if !reflect.DeepEqual(second.Endpoints, first.Endpoints) || len(second.Refused) != 0 {
		t.Fatalf("second assignment = %#v, want %#v", second, first)
	}
}

// Many more than sixty-four ids get distinct, stable numbers and survive a
// second load; no capacity other than the 16-bit endpoint space applies.
func TestStoreAssignsManyMoreThanSixtyFourStableNumbers(t *testing.T) {
	var disk recorder
	store := NewStore()
	const count = 100
	ids := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		ids = append(ids, syntheticID(index))
	}
	first := store.Assign(ids, disk.save)
	if len(first.Endpoints) != count || len(first.Refused) != 0 {
		t.Fatalf("assigning %d ids = %d endpoints, refused %#v", count, len(first.Endpoints), first.Refused)
	}
	for index, id := range ids {
		if first.Endpoints[id] != uint16(index+1) {
			t.Fatalf("%s = %d, want %d", id, first.Endpoints[id], index+1)
		}
	}
	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second := reloaded.Assign(ids, refuseSave(t))
	if !reflect.DeepEqual(second.Endpoints, first.Endpoints) || len(second.Refused) != 0 {
		t.Fatalf("second assignment = %#v, want %#v", second, first)
	}
}

// With one number left and two new ids, the last legal number 0xFFFE is
// assigned and saved, survives a load, and the extra id is refused with the
// exhaustion reason and holds no number.
func TestStoreAssignsTheLastLegalNumberAndRefusesPastIt(t *testing.T) {
	var disk recorder
	store := NewStore()
	const lastLegal = 0xFFFE
	for index := 1; index < lastLegal; index++ {
		id := syntheticID(index)
		store.byID[id] = uint16(index)
		store.byNumber[uint16(index)] = id
	}
	if store.Len() != lastLegal-1 {
		t.Fatalf("filled store = %d ids, want %d", store.Len(), lastLegal-1)
	}
	last, extra := syntheticID(lastLegal), syntheticID(lastLegal+1)
	result := store.Assign([]string{extra, last}, disk.save)
	if len(result.Endpoints) != 1 || result.Endpoints[last] != lastLegal {
		t.Fatalf("endpoints = %#v, want %s at %d", result.Endpoints, last, lastLegal)
	}
	if reason := result.Refused[extra]; reason != ReasonExhausted || len(result.Refused) != 1 {
		t.Fatalf("refused = %#v, want %s for %s", result.Refused, ReasonExhausted, extra)
	}
	if _, ok := store.Endpoint(extra); ok {
		t.Fatal("refused id holds a number")
	}
	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load of the saved document: %v", err)
	}
	if number, ok := reloaded.Endpoint(last); !ok || number != lastLegal {
		t.Fatalf("reloaded %s = %d, %t; want %d", last, number, ok, lastLegal)
	}
	if _, ok := reloaded.Endpoint(extra); ok {
		t.Fatal("refused id was saved")
	}
}

// Several transformed facts of one resource are one endpoint, and an
// unmapped, unsupported, withheld or undisposed element or a relation is
// withheld with its named reason while the resource keeps its endpoint.
func TestOneResourceIsOneEndpointAndEveryOtherDispositionIsWithheld(t *testing.T) {
	rule := mustRule(t, syntheticRows(), exclusionDocument())
	decision := rule.Decide([]Live{
		{ResourceID: resA, Element: power},
		{ResourceID: resA, Element: current},
		{ResourceID: resA, Element: current},
		{ResourceID: resA, Element: unsupported},
		{ResourceID: resB, Element: withheld},
		{ResourceID: resB, Element: blank},
		{ResourceID: resB, Element: relation},
		{ResourceID: resC, Element: unmapped},
		{ResourceID: resC, Element: ""},
		{ResourceID: " " + resD, Element: current},
	})
	wantPublished := []Resource{{ID: resA, Elements: []string{current, power}}}
	if !reflect.DeepEqual(decision.Published, wantPublished) {
		t.Fatalf("published = %#v, want %#v", decision.Published, wantPublished)
	}
	wantWithheld := []Withholding{
		{ResourceID: " " + resD, Element: current, Reason: ReasonResourceID},
		{ResourceID: resA, Element: unsupported, Reason: "element disposition is unsupported"},
		{ResourceID: resB, Element: withheld, Reason: "element disposition is withheld"},
		{ResourceID: resB, Element: blank, Reason: ReasonNoDisposition},
		{ResourceID: resB, Element: relation, Reason: ReasonRelation},
		{ResourceID: resC, Element: "", Reason: ReasonUnmapped},
		{ResourceID: resC, Element: unmapped, Reason: ReasonUnmapped},
	}
	if !reflect.DeepEqual(decision.Withheld, wantWithheld) {
		t.Fatalf("withheld = %#v, want %#v", decision.Withheld, wantWithheld)
	}
	if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA}) {
		t.Fatalf("resource ids = %#v", got)
	}
}

// Each field of a roster row and of a live resource is varied on its own.
// Element: a rule keyed by anything other than the element would publish the
// renamed row. Kind: a rule that ignores the kind would publish the relation.
// Disposition: a rule that treats any row as mapped would publish it.
// ResourceID: a rule that keys the endpoint by element would merge resources.
func TestEveryRowAndLiveFieldDecides(t *testing.T) {
	base := Row{Element: current, Kind: "fact", Disposition: Transformed}
	rows := map[string]Row{
		"element":      {Element: current + ".other", Kind: base.Kind, Disposition: base.Disposition},
		"kind":         {Element: base.Element, Kind: KindRelation, Disposition: base.Disposition},
		"disposition":  {Element: base.Element, Kind: base.Kind, Disposition: Unknown},
		"unrecognised": {Element: base.Element, Kind: base.Kind, Disposition: "proposed"},
	}
	reasons := map[string]string{
		"element":      ReasonUnmapped,
		"kind":         ReasonRelation,
		"disposition":  "element disposition is unknown",
		"unrecognised": "element disposition is not recognised",
	}
	for field, row := range rows {
		t.Run(field, func(t *testing.T) {
			rule := mustRule(t, []Row{row}, exclusionDocument())
			decision := rule.Decide([]Live{{ResourceID: resA, Element: current}})
			if len(decision.Published) != 0 {
				t.Fatalf("published %#v", decision.Published)
			}
			want := []Withholding{{ResourceID: resA, Element: current, Reason: reasons[field]}}
			if !reflect.DeepEqual(decision.Withheld, want) {
				t.Fatalf("withheld = %#v, want %#v", decision.Withheld, want)
			}
		})
	}
	t.Run("resource id", func(t *testing.T) {
		rule := mustRule(t, []Row{base}, exclusionDocument())
		decision := rule.Decide([]Live{{ResourceID: resA, Element: current}, {ResourceID: resB, Element: current}})
		if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA, resB}) {
			t.Fatalf("resource ids = %#v", got)
		}
	})
	t.Run("live element", func(t *testing.T) {
		rule := mustRule(t, []Row{base}, exclusionDocument())
		decision := rule.Decide([]Live{{ResourceID: resA, Element: power}})
		want := []Withholding{{ResourceID: resA, Element: power, Reason: ReasonUnmapped}}
		if len(decision.Published) != 0 || !reflect.DeepEqual(decision.Withheld, want) {
			t.Fatalf("decision = %#v", decision)
		}
	})
}

func TestRuleRefusesAnAmbiguousRoster(t *testing.T) {
	cases := map[string][]Row{
		"duplicate element": {{Element: current, Kind: "fact", Disposition: Transformed}, {Element: current, Kind: "fact", Disposition: Unknown}},
		"empty element":     {{Element: "", Kind: "fact", Disposition: Transformed}},
		"padded element":    {{Element: " " + current, Kind: "fact", Disposition: Transformed}},
	}
	for name, rows := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := newRule(rows, exclusionDocument()); err == nil {
				t.Fatal("ambiguous roster was accepted")
			}
		})
	}
}

func TestCommittedExclusionListIsEmptyAndEmbedded(t *testing.T) {
	onDisk, err := os.ReadFile("exclusions.json")
	if err != nil {
		t.Fatalf("read exclusions.json: %v", err)
	}
	if !bytes.Equal(onDisk, exclusionsDocument) {
		t.Fatal("embedded exclusion list differs from the committed file")
	}
	elements, err := Exclusions()
	if err != nil {
		t.Fatalf("Exclusions: %v", err)
	}
	if len(elements) != 0 {
		t.Fatalf("committed exclusion list = %#v, want empty", elements)
	}
	rule, err := NewRule(syntheticRows())
	if err != nil {
		t.Fatalf("NewRule: %v", err)
	}
	decision := rule.Decide([]Live{{ResourceID: resA, Element: excluded}})
	if got := decision.ResourceIDs(); !reflect.DeepEqual(got, []string{resA}) {
		t.Fatalf("committed rule withheld an element that no list names: %#v", decision)
	}
}

func TestMalformedExclusionListIsRefused(t *testing.T) {
	cases := map[string]string{
		"not an object":     `["x"]`,
		"no elements":       `{}`,
		"null elements":     `{"elements":null}`,
		"extra field":       `{"elements":[],"resources":[]}`,
		"duplicate key":     `{"elements":[],"elements":[]}`,
		"duplicate element": `{"elements":["x","x"]}`,
		"empty element":     `{"elements":[""]}`,
		"padded element":    `{"elements":[" x"]}`,
		"not text":          `{"elements":[1]}`,
		"trailing data":     `{"elements":[]} {}`,
		"empty document":    ``,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseExclusions([]byte(document)); err == nil {
				t.Fatal("malformed exclusion list was accepted")
			}
		})
	}
}

// Absence frees nothing: a resource that leaves and returns keeps its number,
// and a number given once is never given to another id.
func TestAbsentResourceKeepsItsNumberAndNoNumberIsReused(t *testing.T) {
	var disk recorder
	store := NewStore()
	store.Assign([]string{resA, resB}, disk.save)
	onlyB := store.Assign([]string{resB}, refuseSave(t))
	if want := map[string]uint16{resB: 2}; !reflect.DeepEqual(onlyB.Endpoints, want) {
		t.Fatalf("assignment without a = %#v", onlyB.Endpoints)
	}
	withC := store.Assign([]string{resC}, disk.save)
	if want := map[string]uint16{resC: 3}; !reflect.DeepEqual(withC.Endpoints, want) {
		t.Fatalf("a new id reused a number: %#v", withC.Endpoints)
	}
	reloaded, err := Load(disk.last(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	back := reloaded.Assign([]string{resA, resA}, refuseSave(t))
	if want := map[string]uint16{resA: 1}; !reflect.DeepEqual(back.Endpoints, want) {
		t.Fatalf("returning resource = %#v", back.Endpoints)
	}
}

func TestStoreAssignsTheLowestFreeNumber(t *testing.T) {
	store, err := Load([]byte(`{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-b","endpoint":3}]}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var disk recorder
	result := store.Assign([]string{resD, resC}, disk.save)
	if want := map[string]uint16{resC: 2, resD: 4}; !reflect.DeepEqual(result.Endpoints, want) {
		t.Fatalf("assignment = %#v, want %#v", result.Endpoints, want)
	}
	want := `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-c","endpoint":2},{"resource_id":"synthetic-resource-b","endpoint":3},{"resource_id":"synthetic-resource-d","endpoint":4}]}`
	if got := string(disk.last(t)); got != want {
		t.Fatalf("saved = %s, want %s", got, want)
	}
}

func TestFailedSaveExposesNothingNew(t *testing.T) {
	disk := recorder{}
	store := NewStore()
	store.Assign([]string{resA}, disk.save)
	before := store.Document()
	disk.fail = errors.New("synthetic save failure")
	result := store.Assign([]string{resA, resB}, disk.save)
	if want := map[string]uint16{resA: 1}; !reflect.DeepEqual(result.Endpoints, want) {
		t.Fatalf("endpoints = %#v", result.Endpoints)
	}
	if want := map[string]string{resB: ReasonUnsaved}; !reflect.DeepEqual(result.Refused, want) {
		t.Fatalf("refused = %#v", result.Refused)
	}
	if !bytes.Equal(store.Document(), before) {
		t.Fatal("store changed after a failed save")
	}
	noSaver := store.Assign([]string{resB}, nil)
	if want := map[string]string{resB: ReasonUnsaved}; !reflect.DeepEqual(noSaver.Refused, want) {
		t.Fatalf("refused without a saver = %#v", noSaver.Refused)
	}
	disk.fail = nil
	retry := store.Assign([]string{resB}, disk.save)
	if want := map[string]uint16{resB: 2}; !reflect.DeepEqual(retry.Endpoints, want) {
		t.Fatalf("retry = %#v", retry.Endpoints)
	}
}

func TestStoreRefusesAMalformedID(t *testing.T) {
	store := NewStore()
	result := store.Assign([]string{"", " " + resA, resA + "\n"}, refuseSave(t))
	if len(result.Endpoints) != 0 || len(result.Refused) != 3 {
		t.Fatalf("assignment = %#v", result)
	}
	for id, reason := range result.Refused {
		if reason != ReasonResourceID {
			t.Fatalf("%q refused with %q", id, reason)
		}
	}
	if store.Len() != 0 {
		t.Fatal("a malformed id was stored")
	}
}

func TestMalformedStoreDocumentIsRefused(t *testing.T) {
	cases := map[string]string{
		"empty document":    ``,
		"not json":          `{not json`,
		"not an object":     `[]`,
		"no endpoints":      `{}`,
		"null endpoints":    `{"endpoints":null}`,
		"extra field":       `{"endpoints":[],"node":"x"}`,
		"duplicate key":     `{"endpoints":[],"endpoints":[]}`,
		"null record":       `{"endpoints":[null]}`,
		"record extra":      `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1,"element":"x"}]}`,
		"record no id":      `{"endpoints":[{"endpoint":1}]}`,
		"record no number":  `{"endpoints":[{"resource_id":"synthetic-resource-a"}]}`,
		"record dup key":    `{"endpoints":[{"resource_id":"synthetic-resource-a","resource_id":"synthetic-resource-b","endpoint":1}]}`,
		"empty id":          `{"endpoints":[{"resource_id":"","endpoint":1}]}`,
		"padded id":         `{"endpoints":[{"resource_id":" synthetic-resource-a","endpoint":1}]}`,
		"number zero":       `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":0}]}`,
		"number too large":  `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":65535}]}`,
		"number fraction":   `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1.5}]}`,
		"number as text":    `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":"1"}]}`,
		"number as boolean": `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":true}]}`,
		"duplicate id":      `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-a","endpoint":2}]}`,
		"duplicate number":  `{"endpoints":[{"resource_id":"synthetic-resource-a","endpoint":1},{"resource_id":"synthetic-resource-b","endpoint":1}]}`,
		"trailing data":     `{"endpoints":[]}{}`,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			store, err := Load([]byte(document))
			if err == nil || store != nil {
				t.Fatalf("malformed document was accepted: %v", store)
			}
		})
	}
}
