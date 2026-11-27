// Package fixedlabel is the Fixed Label cluster table of Matter 1.6.1: identity,
// attributes and the text an endpoint uses to state that a value comes from
// certification.
package fixedlabel

import (
	"errors"
	"unicode/utf8"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters"
)

// CertificationValue is the Value of a LabelStruct entry that states a
// measurement type's accuracy figures come from certification. The Label of the
// entry is the MeasurementTypeEnum member name of the model.
const CertificationValue = "certification"

// MaxLength is the maximum length of a Label and of a Value (LabelStruct, max 16).
const MaxLength = 16

// ErrLength is returned for a Label or Value longer than MaxLength.
var ErrLength = errors.New("fixed label: text is longer than the model's maximum")

// Entry is one LabelStruct.
type Entry struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Check refuses an entry whose Label or Value exceeds MaxLength characters.
func Check(entry Entry) error {
	if utf8.RuneCountInString(entry.Label) > MaxLength || utf8.RuneCountInString(entry.Value) > MaxLength {
		return ErrLength
	}
	return nil
}

// Table returns a copy of the cluster table.
func Table() clusters.Table {
	return clusters.Table{
		ClusterID: 0x0040, Name: "Fixed Label", Revision: 1,
		Features: []clusters.Feature{},
		Attributes: []clusters.Attribute{
			{ID: 0x0000, Name: "LabelList", MatterType: "list[LabelStruct]", Nullable: false, Conformance: "M", Access: "R V", Quality: "N", Mandatory: true},
			{ID: 0xFFFC, Name: "FeatureMap", MatterType: "bitmap32", Conformance: "M", Access: "R V", Mandatory: true},
			{ID: 0xFFFD, Name: "ClusterRevision", MatterType: "uint16", Conformance: "M", Access: "R V", Mandatory: true},
		},
		Commands: []clusters.Command{},
		Mappings: []clusters.Mapping{},
	}
}
