package projection

import (
	"fmt"

	semreg "github.com/Project-Helianthus/helianthus-semreg/semreg/v1"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/packs/evse"
	semprojection "github.com/Project-Helianthus/helianthus-semreg/semreg/v1/projection"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// ProjectedAttribute is one profile row emitted by the profile-driven projection.
// Value is omitted from the meaning of a null attribute: Null reports that case.
type ProjectedAttribute struct {
	ClusterID   uint32 `json:"cluster_id"`
	AttributeID uint32 `json:"attribute_id"`
	Name        string `json:"name"`
	MatterType  string `json:"matter_type"`
	Nullable    bool   `json:"nullable"`
	Null        bool   `json:"null"`
	Declared    bool   `json:"declared"`
	Value       any    `json:"value"`
	Origin      string `json:"origin"`
	Element     string `json:"element,omitempty"`
	Encoder     string `json:"encoder,omitempty"`
}

// ProfileResult is the profile-driven projection. It is not the one-attribute
// document CanonicalJSON161 checks. Admitted is false while any mandatory row
// has no transformed element and no declared value.
type ProfileResult struct {
	Kind         string                                `json:"kind"`
	Admitted     bool                                  `json:"admitted"`
	Missing      []string                              `json:"missing"`
	Attributes   []ProjectedAttribute                  `json:"attributes"`
	Folds        []composition.Fold                    `json:"folds"`
	Dispositions []semprojection.ProjectionDisposition `json:"dispositions"`
}

// ProjectProfile161 projects the loaded profile for kind from the snapshot.
// Attribute identity, type and source are read from the profile rows.
func ProjectProfile161(kind string, snapshot semreg.Snapshot, evaluation semreg.EvaluationView) (ProfileResult, error) {
	source, err := composition.Load()
	if err != nil {
		return ProfileResult{}, err
	}
	profile, ok := source.Lookup(kind)
	if !ok {
		return ProfileResult{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter profile kind"}
	}
	if err := snapshot.Validate(); err != nil {
		return ProfileResult{}, err
	}
	if err := evaluation.Validate(); err != nil {
		return ProfileResult{}, err
	}
	if err := validateEvaluationCorrespondence(snapshot, evaluation); err != nil {
		return ProfileResult{}, err
	}
	result := ProfileResult{
		Kind: profile.Kind, Missing: []string{}, Attributes: []ProjectedAttribute{},
		Folds: append([]composition.Fold(nil), profile.Folds...), Dispositions: []semprojection.ProjectionDisposition{},
	}
	for _, row := range profile.Attributes {
		if row.Mandatory && row.Origin == composition.OriginMissing {
			result.Missing = append(result.Missing, row.ReportName())
		}
		if row.Field != "" || row.Origin == composition.OriginMissing {
			continue
		}
		switch row.Origin {
		case composition.OriginDeclared:
			result.Attributes = append(result.Attributes, ProjectedAttribute{
				ClusterID: row.ClusterID, AttributeID: row.AttributeID, Name: row.Name, MatterType: row.MatterType,
				Nullable: row.Nullable, Declared: true, Value: row.Declared, Origin: row.Origin,
			})
		case composition.OriginSemreg:
			attribute, disposition, err := projectSemregRow(row, profile.Folds, snapshot, evaluation)
			if err != nil {
				return ProfileResult{}, err
			}
			result.Attributes = append(result.Attributes, attribute)
			if err := disposition.Validate(); err != nil {
				return ProfileResult{}, err
			}
			result.Dispositions = append(result.Dispositions, disposition)
		default:
			return ProfileResult{}, fmt.Errorf("matter profile row %s has no source", row.ReportName())
		}
	}
	result.Admitted = len(result.Missing) == 0
	return result, nil
}

func projectSemregRow(row composition.AttributeRow, folds []composition.Fold, snapshot semreg.Snapshot, evaluation semreg.EvaluationView) (ProjectedAttribute, semprojection.ProjectionDisposition, error) {
	attribute := ProjectedAttribute{
		ClusterID: row.ClusterID, AttributeID: row.AttributeID, Name: row.Name, MatterType: row.MatterType,
		Nullable: row.Nullable, Origin: row.Origin, Element: row.Element, Encoder: row.Encoder,
	}
	unknown := unknownDisposition(row.Element)
	var matched []semreg.FactCandidate
	for _, fact := range snapshot.Facts {
		if !profileKey(row, fact.Key) {
			continue
		}
		if len(fact.Conflicts) != 0 {
			return ProjectedAttribute{}, semprojection.ProjectionDisposition{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter projection conflict"}
		}
		for _, candidate := range fact.Candidates {
			if !profileKey(row, candidate.Key) {
				continue
			}
			matched = append(matched, candidate)
		}
	}
	var good []semreg.FactCandidate
	for _, candidate := range matched {
		if !usableCandidate(candidate) || !evaluationUsable(candidate.CandidateID, evaluation) {
			continue
		}
		if row.Element == "evse.ac.current" {
			if err := evse.New().ValidateFact(candidate.Key, candidate.Value); err != nil {
				return ProjectedAttribute{}, semprojection.ProjectionDisposition{}, err
			}
		}
		good = append(good, candidate)
	}
	if len(good) == 0 {
		if row.Nullable {
			attribute.Null = true
		}
		return attribute, unknown, nil
	}
	if len(good) > 1 {
		return ProjectedAttribute{}, semprojection.ProjectionDisposition{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter projection ambiguous current"}
	}
	if len(matched) != 1 {
		return ProjectedAttribute{}, semprojection.ProjectionDisposition{}, &semreg.Error{ID: semreg.PreconditionFailed, Detail: "matter projection ambiguous current"}
	}
	encoded, err := encodeRow(row, *good[0].Value)
	if err != nil {
		return ProjectedAttribute{}, semprojection.ProjectionDisposition{}, err
	}
	attribute.Value = encoded
	reason := semreg.DefinitionID("matter.phase_endpoint_identity_loss.v1")
	loss := []semprojection.LossDetail{}
	for _, fold := range folds {
		if !fold.Declared {
			continue
		}
		items := make([]semreg.DefinitionID, len(fold.SourceItems))
		for i, item := range fold.SourceItems {
			items[i] = semreg.DefinitionID(item)
		}
		loss = append(loss, semprojection.LossDetail{Kind: semprojection.LossKind(fold.Kind), SourceItems: items, Description: fold.Description})
	}
	loss = append(loss, semprojection.LossDetail{
		Kind: semprojection.LossUnit, SourceItems: []semreg.DefinitionID{semreg.DefinitionID(row.Element)},
		Description: "amperes are represented as integral milliamperes only when exact and representable",
	})
	return attribute, semprojection.ProjectionDisposition{
		Kind: semprojection.ItemFact, ItemID: semreg.DefinitionID(row.Element), Outcome: semprojection.ProjectionTransformed,
		SourceKeys: []semreg.FactKey{good[0].Key}, Loss: loss, Reason: &reason,
	}, nil
}

func encodeRow(row composition.AttributeRow, value semreg.Value) (string, error) {
	switch row.Encoder {
	case "ampere_to_milliampere":
		return amperesToMilliampere(value)
	default:
		return "", fmt.Errorf("matter profile encoder %s", row.Encoder)
	}
}

func profileKey(row composition.AttributeRow, key semreg.FactKey) bool {
	if string(key.PackID) != row.PackID || string(key.PackVersion) != row.PackVersion || string(key.FactID) != row.Element {
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

func evaluationUsable(id semreg.CandidateID, evaluation semreg.EvaluationView) bool {
	for _, item := range evaluation.Facts {
		if item.CandidateID != id {
			continue
		}
		return item.Freshness == semreg.FreshnessFresh && item.EffectiveAvailability == semreg.AvailabilityAvailable
	}
	return false
}

func usableCandidate(candidate semreg.FactCandidate) bool {
	quality := candidate.Quality
	return quality.Assertion == semreg.AssertionObserved && quality.Qualification == semreg.QualificationQualified && quality.Promotion == semreg.PromotionPromoted && quality.Validity == semreg.ValidityGood && quality.Availability == semreg.AvailabilityAvailable && quality.Freshness == semreg.FreshnessFresh && candidate.Value != nil
}

func unknownDisposition(element string) semprojection.ProjectionDisposition {
	reason := semreg.DefinitionID("matter.unmapped.v1")
	return semprojection.ProjectionDisposition{
		Kind: semprojection.ItemFact, ItemID: semreg.DefinitionID(element), Outcome: semprojection.ProjectionUnknown,
		SourceKeys: []semreg.FactKey{}, Loss: []semprojection.LossDetail{}, Reason: &reason,
	}
}
