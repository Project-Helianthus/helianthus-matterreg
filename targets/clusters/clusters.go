// Package clusters holds the row types shared by the cluster tables under
// this directory. Each subdirectory is one cluster and imports only this file.
package clusters

// Feature is one bit of a cluster's FeatureMap.
type Feature struct {
	Bit         uint
	Code        string
	Name        string
	Conformance string
}

// Attribute is one attribute of the data model. Mandatory is true only for an
// unconditional M conformance; every other conformance is kept as text.
type Attribute struct {
	ID          uint32
	Name        string
	MatterType  string
	Nullable    bool
	Conformance string
	Access      string
	Quality     string
	Mandatory   bool
}

// Command is one command of the data model. A command with no mapping has no
// SemReg operation and no inverse encoder.
type Command struct {
	ID          uint32
	Name        string
	Direction   string
	Conformance string
}

// Mapping is the one transformed row of the target for an attribute: the
// SemReg element that feeds it and the exact conversion the target states.
type Mapping struct {
	AttributeID     uint32
	PackID          string
	PackVersion     string
	Element         string
	Dimension       string
	Encoder         string
	Reason          string
	FromUnit        string
	ToUnit          string
	Factor          int64
	Offset          int64
	Rounding        string
	RequireIntegral bool
	Minimum         string
	Maximum         string
}

// Table is one cluster: identity, features, attributes, commands and the
// attributes the target maps.
type Table struct {
	ClusterID  uint32
	Name       string
	Revision   uint16
	Features   []Feature
	Attributes []Attribute
	Commands   []Command
	Mappings   []Mapping
}
