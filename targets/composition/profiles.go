// Package composition loads the accepted pilot profile for the factory.
// One source is keyed by resource kind. Callers receive a copy.
package composition

import (
	"fmt"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

// Origin values match the pilot table.
const (
	OriginSemreg   = string(pilot.OriginSemreg)
	OriginDeclared = string(pilot.OriginDeclared)
	OriginMissing  = string(pilot.OriginMissing)
)

// CommandRow is one mapped attribute or request command.
// PartKey is empty for the root endpoint.
type CommandRow struct {
	PartKey        string
	ClusterID      uint32
	ElementID      uint32
	Operation      string
	Capability     string
	InverseEncoder string
}

// ServerRow is one cluster served at a part.
type ServerRow struct {
	ClusterID uint32
	Name      string
	Revision  uint32
}

// AttributeRow is one typed attribute source copied from the pilot table.
type AttributeRow struct {
	PartKey     string
	ClusterID   uint32
	ClusterName string
	AttributeID uint32
	Name        string
	MatterType  string
	Nullable    bool
	Mandatory   bool
	Field       string
	Origin      string
	PackID      string
	PackVersion string
	Element     string
	Dimension   string
	Encoder     string
	Reason      string
	Declared    any
	Detail      string
}

// ReportName names a missing mandatory row.
func (r AttributeRow) ReportName() string {
	name := r.ClusterName + "." + r.Name
	if r.Field != "" {
		name += "." + r.Field
	}
	return name
}

// PartSpec is one composed part and the rows that belong to it.
type PartSpec struct {
	Role         string
	DeviceTypeID uint32
	Revision     uint32
	Count        int
	Children     []PartSpec
	Attributes   []AttributeRow
	Servers      []ServerRow
}

// Fold is a declared loss of a component into the endpoint.
type Fold struct {
	Name        string
	Kind        string
	SourceItems []string
	Description string
	Declared    bool
}

// Profile is the accepted document and its typed rows for one resource kind.
type Profile struct {
	Kind       string
	PartKey    string
	Document   []byte
	Attributes []AttributeRow
	Commands   []CommandRow
	Parts      []PartSpec
	Folds      []Fold
}

// Source is the single kind index. Lookup of an unknown kind is false.
type Source struct {
	byKind map[string]Profile
}

// Load reads the embedded accepted pilot profile. It does not open a path and
// it does not consult another index.
func Load() (Source, error) {
	profile, err := pilotProfile()
	if err != nil {
		return Source{}, err
	}
	return sourceFrom([]Profile{profile})
}

// Lookup returns a copy of the profile for kind.
func (s Source) Lookup(kind string) (Profile, bool) {
	profile, ok := s.byKind[kind]
	if !ok {
		return Profile{}, false
	}
	return cloneProfile(profile), true
}

func sourceFrom(profiles []Profile) (Source, error) {
	source := Source{byKind: make(map[string]Profile, len(profiles))}
	for _, profile := range profiles {
		if err := validateProfile(profile); err != nil {
			return Source{}, err
		}
		if profile.Kind == "" {
			return Source{}, fmt.Errorf("composition profile kind is empty")
		}
		if _, exists := source.byKind[profile.Kind]; exists {
			return Source{}, fmt.Errorf("composition profile kind %s is duplicated", profile.Kind)
		}
		source.byKind[profile.Kind] = profile
	}
	return source, nil
}

func validateProfile(profile Profile) error {
	roles := map[string]struct{}{}
	var walk func([]PartSpec) error
	walk = func(parts []PartSpec) error {
		for _, part := range parts {
			if part.Role == "" {
				return fmt.Errorf("composition part role is empty")
			}
			if _, exists := roles[part.Role]; exists {
				return fmt.Errorf("composition part role %s is duplicated", part.Role)
			}
			roles[part.Role] = struct{}{}
			if err := walk(part.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(profile.Parts); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, command := range profile.Commands {
		if command.PartKey != "" {
			if _, ok := roles[command.PartKey]; !ok {
				return fmt.Errorf("composition command part %s is not a part role", command.PartKey)
			}
		}
		key := fmt.Sprintf("%s\x00%d\x00%d", command.PartKey, command.ClusterID, command.ElementID)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("composition command selector %s %d %d is duplicated", command.PartKey, command.ClusterID, command.ElementID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func pilotProfile() (Profile, error) {
	rows := pilot.Rows()
	attributes := make([]AttributeRow, len(rows))
	for i, row := range rows {
		attributes[i] = AttributeRow{
			ClusterID: row.ClusterID, ClusterName: row.ClusterName, AttributeID: row.AttributeID,
			Name: row.Name, MatterType: row.MatterType, Nullable: row.Nullable, Mandatory: row.Mandatory,
			Field: row.Field, Origin: string(row.Origin), PackID: row.PackID, PackVersion: row.PackVersion,
			Element: row.Element, Dimension: row.Dimension, Encoder: row.Encoder, Reason: row.Reason, Declared: row.Declared, Detail: row.Detail,
		}
	}
	folds := pilot.Folds()
	copied := make([]Fold, len(folds))
	for i, fold := range folds {
		copied[i] = Fold{Name: fold.Name, Kind: fold.Kind, SourceItems: append([]string(nil), fold.SourceItems...), Description: fold.Description, Declared: fold.Declared}
	}
	profile := Profile{
		Kind: pilot.ResourceKind, Document: pilot.Document(), Attributes: attributes,
		Commands: []CommandRow{}, Parts: []PartSpec{}, Folds: copied,
	}
	return profile, nil
}

func cloneProfile(profile Profile) Profile {
	out := profile
	out.Document = append([]byte(nil), profile.Document...)
	out.Attributes = cloneAttributes(profile.Attributes)
	out.Commands = append([]CommandRow(nil), profile.Commands...)
	out.Parts = cloneParts(profile.Parts)
	out.Folds = cloneFolds(profile.Folds)
	return out
}

func cloneAttributes(rows []AttributeRow) []AttributeRow {
	out := append([]AttributeRow(nil), rows...)
	for i := range out {
		out[i].Declared = cloneValue(rows[i].Declared)
	}
	return out
}

func cloneParts(parts []PartSpec) []PartSpec {
	out := append([]PartSpec(nil), parts...)
	for i := range out {
		out[i].Children = cloneParts(parts[i].Children)
		out[i].Attributes = cloneAttributes(parts[i].Attributes)
		out[i].Servers = append([]ServerRow(nil), parts[i].Servers...)
	}
	return out
}

func cloneFolds(folds []Fold) []Fold {
	out := append([]Fold(nil), folds...)
	for i := range out {
		out[i].SourceItems = append([]string(nil), folds[i].SourceItems...)
	}
	return out
}

func cloneValue(v any) any {
	switch n := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(n))
		for key, item := range n {
			out[key] = cloneValue(item)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i := range n {
			out[i] = cloneValue(n[i])
		}
		return out
	default:
		return v
	}
}
