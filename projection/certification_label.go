package projection

import (
	"fmt"
	"sort"
	"strings"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/electricalpowermeasurement"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/fixedlabel"
)

// Source of an accuracy entry. A measurement type whose figures the device
// publishes is SourceDevice and carries no statement; a type whose figures come
// from a fact of declared origin with certification evidence is
// SourceCertification and is named in the Fixed Label list.
const (
	SourceDevice        = "device"
	SourceCertification = "certification"
)

const (
	accuracyClusterName       = "Electrical Power Measurement"
	accuracyEnergyClusterName = "Electrical Energy Measurement"
	accuracyAttributeID       = 0x0002
	powerMeasurementClusterID = 0x0090
	accuracyPackPrefix        = "helianthus.pack."
	accuracyTypeDimension     = ".dimension.measurement_type"
)

type accuracyMember string

const (
	memberMeasured   accuracyMember = "Measured"
	memberMinMeasure accuracyMember = "MinMeasuredValue"
	memberMaxMeasure accuracyMember = "MaxMeasuredValue"
	memberRangeMin   accuracyMember = "RangeMin"
	memberRangeMax   accuracyMember = "RangeMax"
	memberPercentMax accuracyMember = "PercentMax"
)

type accuracyElement struct {
	suffix string
	member accuracyMember
	// typeSymbol is the measurement type the element's own id names; empty when
	// the id names none and the dimension alone selects the type.
	typeSymbol string
}

var accuracyPacks = []string{"evse", "infrastructure", "pv", "storage"}

var accuracyElements = []accuracyElement{
	{"accuracy.measured", memberMeasured, ""},
	{"accuracy.ratio_maximum", memberPercentMax, ""},
	{"accuracy.measured_minimum_active_current", memberMinMeasure, "active_current"},
	{"accuracy.measured_minimum_active_power", memberMinMeasure, "active_power"},
	{"accuracy.measured_maximum_active_current", memberMaxMeasure, "active_current"},
	{"accuracy.measured_maximum_active_power", memberMaxMeasure, "active_power"},
	{"accuracy.range_minimum_active_current", memberRangeMin, "active_current"},
	{"accuracy.range_minimum_active_power", memberRangeMin, "active_power"},
	{"accuracy.range_maximum_active_current", memberRangeMax, "active_current"},
	{"accuracy.range_maximum_active_power", memberRangeMax, "active_power"},
}

var accuracyElementByFact = func() map[string]accuracyElement {
	out := map[string]accuracyElement{}
	for _, pack := range accuracyPacks {
		for _, element := range accuracyElements {
			out[pack+"."+element.suffix] = element
		}
	}
	return out
}()

// AccuracyRange is the one MeasurementAccuracyRangeStruct a source states. A
// member no fact states is nil.
type AccuracyRange struct {
	RangeMin   *string `json:"range_min,omitempty"`
	RangeMax   *string `json:"range_max,omitempty"`
	PercentMax *uint16 `json:"percent_max,omitempty"`
}

// AccuracyEntry is one MeasurementAccuracyStruct. Missing names every mandatory
// member no fact states; none is completed with a default.
type AccuracyEntry struct {
	MeasurementType   string        `json:"measurement_type"`
	MeasurementTypeID uint16        `json:"measurement_type_id"`
	ClusterID         uint32        `json:"cluster_id"`
	Source            string        `json:"source"`
	Measured          *bool         `json:"measured,omitempty"`
	MinMeasuredValue  *string       `json:"min_measured_value,omitempty"`
	MaxMeasuredValue  *string       `json:"max_measured_value,omitempty"`
	Range             AccuracyRange `json:"accuracy_range"`
	Missing           []string      `json:"missing"`
}

// AccuracyRefusal names a figure that was not emitted and why.
type AccuracyRefusal struct {
	Element         string `json:"element"`
	MeasurementType string `json:"measurement_type"`
	Reason          string `json:"reason"`
}

// AccuracyResult is the Accuracy projection of one endpoint. Power is the list
// of Electrical Power Measurement Accuracy; Energy is the single Electrical
// Energy Measurement Accuracy struct. FixedLabel is nil unless at least one
// entry comes from certification.
type AccuracyResult struct {
	Power      []AccuracyEntry    `json:"power"`
	Energy     *AccuracyEntry     `json:"energy,omitempty"`
	FixedLabel []fixedlabel.Entry `json:"fixed_label,omitempty"`
	Refusals   []AccuracyRefusal  `json:"refusals"`
}

// Empty reports whether the projection holds no entry and no refusal.
func (r AccuracyResult) Empty() bool {
	return len(r.Power) == 0 && r.Energy == nil && len(r.Refusals) == 0
}

type accuracyBucket struct {
	device map[accuracyMember]accuracyFigure
	cert   map[accuracyMember]accuracyFigure
}

type accuracyFigure struct {
	element string
	value   semreg.Value
}

// ProjectAccuracy161 projects the accuracy facts of the snapshot. The source of
// a measurement type is read from the kernel origin of its candidates: a
// candidate of declared origin with certification evidence is a certification
// figure and a usable observation is a device figure. When the device publishes
// any figure of a measurement type, only device figures are used for it.
func ProjectAccuracy161(snapshot semreg.Snapshot, evaluation semreg.EvaluationView) (AccuracyResult, error) {
	if err := snapshot.Validate(); err != nil {
		return AccuracyResult{}, err
	}
	if err := evaluation.Validate(); err != nil {
		return AccuracyResult{}, err
	}
	if err := validateEvaluationCorrespondence(snapshot, evaluation); err != nil {
		return AccuracyResult{}, err
	}
	return projectAccuracy(snapshot, evaluation)
}

func projectAccuracy(snapshot semreg.Snapshot, evaluation semreg.EvaluationView) (AccuracyResult, error) {
	types := map[string]electricalpowermeasurement.MeasurementType{}
	for _, row := range electricalpowermeasurement.MeasurementTypes() {
		types[row.Symbol] = row
	}
	buckets := map[string]*accuracyBucket{}
	result := AccuracyResult{Power: []AccuracyEntry{}, Refusals: []AccuracyRefusal{}}
	for _, fact := range snapshot.Facts {
		element, symbol, ok := accuracyKey(fact.Key)
		if !ok {
			continue
		}
		if len(fact.Conflicts) != 0 {
			return AccuracyResult{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter projection conflict"}
		}
		row, known := types[symbol]
		if !known {
			result.Refusals = append(result.Refusals, AccuracyRefusal{Element: string(fact.Key.FactID), MeasurementType: symbol, Reason: "measurement type is not a member of the model"})
			continue
		}
		if element.typeSymbol != "" && element.typeSymbol != symbol {
			result.Refusals = append(result.Refusals, AccuracyRefusal{Element: string(fact.Key.FactID), MeasurementType: row.Name, Reason: "element names another measurement type than its dimension"})
			continue
		}
		for _, candidate := range fact.Candidates {
			source := accuracySource(candidate, evaluation)
			if source == "" {
				continue
			}
			validator, ok := profilePackValidators[string(candidate.Key.PackID)]
			if !ok {
				return AccuracyResult{}, fmt.Errorf("matter profile pack %s has no validator", candidate.Key.PackID)
			}
			if err := validator.ValidateFact(candidate.Key, candidate.Value); err != nil {
				return AccuracyResult{}, err
			}
			bucket := buckets[symbol]
			if bucket == nil {
				bucket = &accuracyBucket{device: map[accuracyMember]accuracyFigure{}, cert: map[accuracyMember]accuracyFigure{}}
				buckets[symbol] = bucket
			}
			figures := bucket.cert
			if source == SourceDevice {
				figures = bucket.device
			}
			if _, duplicate := figures[element.member]; duplicate {
				return AccuracyResult{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter projection ambiguous current"}
			}
			figures[element.member] = accuracyFigure{element: string(candidate.Key.FactID), value: *candidate.Value}
		}
	}
	symbols := make([]string, 0, len(buckets))
	for symbol := range buckets {
		symbols = append(symbols, symbol)
	}
	sort.Slice(symbols, func(i, j int) bool { return types[symbols[i]].ID < types[symbols[j]].ID })
	for _, symbol := range symbols {
		row, bucket := types[symbol], buckets[symbol]
		source, figures := SourceCertification, bucket.cert
		if len(bucket.device) != 0 {
			source, figures = SourceDevice, bucket.device
		}
		entry, refusals := buildAccuracyEntry(row, source, figures)
		result.Refusals = append(result.Refusals, refusals...)
		if !entryHasFigure(entry) {
			continue
		}
		switch row.ClusterID {
		case powerMeasurementClusterID:
			result.Power = append(result.Power, entry)
		case electricalpowermeasurement.ElectricalEnergyMeasurementClusterID:
			result.Energy = &entry
		default:
			result.Refusals = append(result.Refusals, AccuracyRefusal{Element: "accuracy", MeasurementType: row.Name, Reason: "no Accuracy attribute carries this measurement type"})
			continue
		}
		if source == SourceCertification {
			label := fixedlabel.Entry{Label: row.Name, Value: fixedlabel.CertificationValue}
			if err := fixedlabel.Check(label); err != nil {
				return AccuracyResult{}, err
			}
			result.FixedLabel = append(result.FixedLabel, label)
		}
	}
	return result, nil
}

func accuracyKey(key semreg.FactKey) (accuracyElement, string, bool) {
	element, ok := accuracyElementByFact[string(key.FactID)]
	if !ok {
		return accuracyElement{}, "", false
	}
	pack := strings.TrimPrefix(string(key.PackID), accuracyPackPrefix)
	if !strings.HasPrefix(string(key.PackID), accuracyPackPrefix) || !strings.HasPrefix(string(key.FactID), pack+".") {
		return accuracyElement{}, "", false
	}
	for _, dimension := range key.Dimensions {
		if string(dimension.ID) == pack+accuracyTypeDimension && dimension.Value.Symbol != nil {
			return element, dimension.Value.Symbol.Token, true
		}
	}
	return accuracyElement{}, "", false
}

// accuracySource classifies a usable candidate: SourceDevice for a native
// observation, SourceCertification for a declared origin with certification
// evidence, and empty for anything else.
func accuracySource(candidate semreg.FactCandidate, evaluation semreg.EvaluationView) string {
	if candidate.Value == nil || !evaluationUsable(candidate.CandidateID, evaluation) {
		return ""
	}
	quality := candidate.Quality
	if quality.Qualification != semreg.QualificationQualified || quality.Promotion != semreg.PromotionPromoted || quality.Validity != semreg.ValidityGood || quality.Availability != semreg.AvailabilityAvailable || quality.Freshness != semreg.FreshnessFresh {
		return ""
	}
	if candidate.Origin.Kind == semreg.OriginDeclared {
		for _, evidence := range candidate.Origin.Evidence {
			if evidence.Kind == semreg.EvidenceKindCertification {
				return SourceCertification
			}
		}
		return ""
	}
	if usableCandidate(candidate) {
		return SourceDevice
	}
	return ""
}

func buildAccuracyEntry(row electricalpowermeasurement.MeasurementType, source string, figures map[accuracyMember]accuracyFigure) (AccuracyEntry, []AccuracyRefusal) {
	entry := AccuracyEntry{MeasurementType: row.Name, MeasurementTypeID: row.ID, ClusterID: row.ClusterID, Source: source, Missing: []string{}}
	var refusals []AccuracyRefusal
	refuse := func(figure accuracyFigure, err error) {
		refusals = append(refusals, AccuracyRefusal{Element: figure.element, MeasurementType: row.Name, Reason: err.Error()})
	}
	if figure, ok := figures[memberMeasured]; ok {
		if figure.value.Kind == semreg.ValueBoolean && figure.value.Boolean != nil {
			measured := *figure.value.Boolean
			entry.Measured = &measured
		} else {
			refuse(figure, fmt.Errorf("value is not a boolean"))
		}
	}
	if figure, ok := figures[memberPercentMax]; ok {
		if quantity := figure.value.Quantity; figure.value.Kind == semreg.ValueQuantity && quantity != nil && quantity.Unit == "unit.ratio" {
			percent, err := electricalpowermeasurement.RatioToPercent100ths(quantity.Number.Coefficient, int(quantity.Number.Exponent10))
			if err != nil {
				refuse(figure, err)
			} else {
				entry.Range.PercentMax = &percent
			}
		} else {
			refuse(figure, fmt.Errorf("value is not a ratio"))
		}
	}
	scaled := func(member accuracyMember) *string {
		figure, ok := figures[member]
		if !ok {
			return nil
		}
		quantity := figure.value.Quantity
		if figure.value.Kind != semreg.ValueQuantity || quantity == nil || row.SemregUnit == "" || string(quantity.Unit) != row.SemregUnit {
			refuse(figure, fmt.Errorf("value is not a quantity in the unit of the measurement type"))
			return nil
		}
		text, err := electricalpowermeasurement.ToMilli(quantity.Number.Coefficient, int(quantity.Number.Exponent10))
		if err != nil {
			refuse(figure, err)
			return nil
		}
		return &text
	}
	entry.MinMeasuredValue = scaled(memberMinMeasure)
	entry.MaxMeasuredValue = scaled(memberMaxMeasure)
	entry.Range.RangeMin = scaled(memberRangeMin)
	entry.Range.RangeMax = scaled(memberRangeMax)
	clusterName := accuracyClusterName
	if row.ClusterID == electricalpowermeasurement.ElectricalEnergyMeasurementClusterID {
		clusterName = accuracyEnergyClusterName
	}
	for _, field := range []struct {
		name    accuracyMember
		present bool
	}{
		{memberMeasured, entry.Measured != nil},
		{memberMinMeasure, entry.MinMeasuredValue != nil},
		{memberMaxMeasure, entry.MaxMeasuredValue != nil},
		{memberRangeMin, entry.Range.RangeMin != nil},
		{memberRangeMax, entry.Range.RangeMax != nil},
	} {
		if !field.present {
			entry.Missing = append(entry.Missing, clusterName+".Accuracy."+string(field.name))
		}
	}
	return entry, refusals
}

func entryHasFigure(entry AccuracyEntry) bool {
	return entry.Measured != nil || entry.MinMeasuredValue != nil || entry.MaxMeasuredValue != nil || entry.Range.RangeMin != nil || entry.Range.RangeMax != nil || entry.Range.PercentMax != nil
}
