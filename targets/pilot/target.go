package pilot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// SourceRow is one target row that gives a mandatory value a SemReg element and
// its encoder. ClusterID, AttributeID and Field select the profile row.
type SourceRow struct {
	ClusterID   uint32 `json:"cluster_id"`
	AttributeID uint32 `json:"attribute_id"`
	Field       string `json:"field"`
	PackID      string `json:"pack_id"`
	PackVersion string `json:"pack_version"`
	Element     string `json:"element"`
	Dimension   string `json:"dimension"`
	Encoder     string `json:"encoder"`
}

// Target is the part of a target document the pilot evaluation reads: the
// source rows it carries for mandatory values. A value without a row is a gap.
type Target struct {
	Sources []SourceRow
}

// Decision is the result of evaluating the profile against a target. Permitted
// is true only when no mandatory row is missing; Missing names every missing
// row and Reasons gives the reason for each, in the same order.
type Decision struct {
	Permitted bool
	Missing   []string
	Reasons   []string
	Rows      []AttributeRow
}

const noSourceReason = "the target carries no source row for this value"

// CommittedTarget is the committed target: it sources no mandatory value of
// the pilot, so every row the profile holds as missing stays a named gap.
func CommittedTarget() Target {
	return Target{}
}

// LoadTarget reads a target document. The document must say in its first field
// that it is synthetic test data; no production path calls this function.
func LoadTarget(raw []byte) (Target, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document struct {
		SyntheticTestData string      `json:"synthetic_test_data"`
		Sources           []SourceRow `json:"sources"`
	}
	if err := decoder.Decode(&document); err != nil {
		return Target{}, err
	}
	if document.SyntheticTestData == "" {
		return Target{}, errors.New("target document does not declare itself synthetic test data")
	}
	if first := bytes.Index(raw, []byte(`"`)); first < 0 || !bytes.HasPrefix(raw[first:], []byte(`"synthetic_test_data"`)) {
		return Target{}, errors.New("synthetic_test_data is not the first field")
	}
	for _, source := range document.Sources {
		if source.PackID == "" || source.PackVersion == "" || source.Element == "" || source.Encoder == "" {
			return Target{}, fmt.Errorf("source row %d.%d %q is incomplete", source.ClusterID, source.AttributeID, source.Field)
		}
	}
	return Target{Sources: document.Sources}, nil
}

// Evaluate gives each missing mandatory row the source the target carries for
// it. A row without one is named and no endpoint is composed: Rows is nil.
func Evaluate(target Target) Decision {
	var decision Decision
	rows := Rows()
	for i, row := range rows {
		if row.Origin != OriginMissing {
			continue
		}
		source, ok := target.source(row)
		if !ok {
			if row.Mandatory {
				decision.Missing = append(decision.Missing, row.ReportName())
				reason := row.Detail
				if reason == "" {
					reason = noSourceReason
				}
				decision.Reasons = append(decision.Reasons, reason)
			}
			continue
		}
		rows[i].Origin = OriginSemreg
		rows[i].PackID, rows[i].PackVersion = source.PackID, source.PackVersion
		rows[i].Element, rows[i].Dimension, rows[i].Encoder = source.Element, source.Dimension, source.Encoder
		rows[i].Detail = ""
	}
	decision.Permitted = len(decision.Missing) == 0
	if decision.Permitted {
		decision.Rows = rows
	}
	return decision
}

func (t Target) source(row AttributeRow) (SourceRow, bool) {
	for _, source := range t.Sources {
		if source.ClusterID == row.ClusterID && source.AttributeID == row.AttributeID && source.Field == row.Field {
			return source, true
		}
	}
	return SourceRow{}, false
}
