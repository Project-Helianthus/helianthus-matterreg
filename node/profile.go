package node

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	clusterDescriptor      uint32 = 0x001D
	clusterBridgedBasicInf uint32 = 0x0039
	attrDeviceTypeList     uint32 = 0x0000
	attrServerList         uint32 = 0x0001
	attrFeatureMap         uint32 = 0xFFFC
	attrClusterRevision    uint32 = 0xFFFD
	attrFirstGlobal        uint32 = 0xFFF8
)

// Selector names one served attribute of one part. PartRole is empty for the
// root endpoint.
type Selector struct {
	PartRole    string
	ClusterID   uint32
	AttributeID uint32
}

// Value is the encoded value of one attribute. Null states the null value of a
// nullable attribute. An attribute with no Value in a resource's Values is
// unknown.
type Value struct {
	Null  bool
	Value any
}

// partRows is the rows of one part (the root when Role is empty) after the
// admission of the resource's elements.
type partRows struct {
	Role         string
	DeviceTypeID uint32
	Servers      []uint32
	Features     map[uint32]uint32
	HasFeatures  map[uint32]bool
	Declared     map[uint32]map[uint32]any
	Dynamic      []dynamicRow
	Missing      []string
	Rows         []string
	Children     []*partRows
}

type dynamicRow struct {
	Selector
	Row composition.AttributeRow
}

// classify walks the profile for one resource and its admitted elements.
func classify(profile composition.Profile, admitted map[string]struct{}) (*partRows, error) {
	root := &partRows{}
	typeID, servers, err := rootIdentity(profile.Attributes)
	if err != nil {
		return nil, err
	}
	root.DeviceTypeID = typeID
	if err := fill(root, servers, profile.Attributes, admitted); err != nil {
		return nil, err
	}
	for _, part := range profile.Parts {
		child, err := classifyPart(part, admitted)
		if err != nil {
			return nil, err
		}
		root.Children = append(root.Children, child)
	}
	return root, nil
}

func classifyPart(part composition.PartSpec, admitted map[string]struct{}) (*partRows, error) {
	if part.Count > 1 {
		return nil, fmt.Errorf("part %s is repeated %d times and the profile states no source-backed role for a repeat", part.Role, part.Count)
	}
	node := &partRows{Role: part.Role, DeviceTypeID: part.DeviceTypeID}
	servers := make([]uint32, 0, len(part.Servers))
	for _, server := range part.Servers {
		servers = append(servers, server.ClusterID)
	}
	if err := fill(node, servers, part.Attributes, admitted); err != nil {
		return nil, err
	}
	for _, child := range part.Children {
		c, err := classifyPart(child, admitted)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, c)
	}
	return node, nil
}

func rootIdentity(rows []composition.AttributeRow) (uint32, []uint32, error) {
	var typeID uint32
	var servers []uint32
	for _, row := range rows {
		if row.ClusterID != clusterDescriptor || row.Origin != composition.OriginDeclared {
			continue
		}
		switch row.AttributeID {
		case attrDeviceTypeList:
			list, ok := row.Declared.([]any)
			if !ok || len(list) == 0 {
				return 0, nil, fmt.Errorf("Descriptor.DeviceTypeList declares no device type")
			}
			entry, ok := list[0].(map[string]any)
			if !ok {
				return 0, nil, fmt.Errorf("Descriptor.DeviceTypeList entry is not a device type")
			}
			id, err := parseID(entry["device_type"])
			if err != nil {
				return 0, nil, fmt.Errorf("Descriptor.DeviceTypeList: %w", err)
			}
			typeID = id
		case attrServerList:
			list, ok := row.Declared.([]any)
			if !ok {
				return 0, nil, fmt.Errorf("Descriptor.ServerList is not a list")
			}
			for _, item := range list {
				id, err := parseID(item)
				if err != nil {
					return 0, nil, fmt.Errorf("Descriptor.ServerList: %w", err)
				}
				servers = append(servers, id)
			}
		}
	}
	if typeID == 0 {
		return 0, nil, fmt.Errorf("profile declares no root device type")
	}
	return typeID, servers, nil
}

func parseID(v any) (uint32, error) {
	switch n := v.(type) {
	case string:
		id, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(n), "0x"), 16, 32)
		if err != nil {
			return 0, fmt.Errorf("identifier %q is not hexadecimal", n)
		}
		return uint32(id), nil
	default:
		return 0, fmt.Errorf("identifier %v is not a string", v)
	}
}

func fill(p *partRows, servers []uint32, rows []composition.AttributeRow, admitted map[string]struct{}) error {
	p.Features = map[uint32]uint32{}
	p.HasFeatures = map[uint32]bool{}
	p.Declared = map[uint32]map[uint32]any{}
	clusters := map[uint32]struct{}{}
	for _, id := range servers {
		clusters[id] = struct{}{}
	}
	for _, row := range rows {
		if row.ClusterID == clusterDescriptor || row.ClusterID == clusterBridgedBasicInf {
			continue
		}
		clusters[row.ClusterID] = struct{}{}
		if row.Mandatory || row.Origin != composition.OriginMissing {
			p.Rows = append(p.Rows, row.ReportName())
		}
		if err := place(p, row, admitted); err != nil {
			return err
		}
	}
	for id := range clusters {
		if id != clusterDescriptor && id != clusterBridgedBasicInf {
			p.Servers = append(p.Servers, id)
		}
	}
	sort.Slice(p.Servers, func(i, j int) bool { return p.Servers[i] < p.Servers[j] })
	sort.Strings(p.Missing)
	return nil
}

func place(p *partRows, row composition.AttributeRow, admitted map[string]struct{}) error {
	if row.AttributeID == attrClusterRevision || (row.AttributeID >= attrFirstGlobal && row.AttributeID < attrFeatureMap) {
		return nil
	}
	switch row.Origin {
	case composition.OriginDeclared:
		if row.Field != "" {
			return nil
		}
		if row.AttributeID == attrFeatureMap {
			features, err := toUint(row.Declared)
			if err != nil || features > math.MaxUint32 {
				return fmt.Errorf("%s declares an invalid feature map", row.ReportName())
			}
			p.Features[row.ClusterID], p.HasFeatures[row.ClusterID] = uint32(features), true
			return nil
		}
		value, err := convert(row.MatterType, row.Declared)
		if err != nil {
			return fmt.Errorf("%s: %w", row.ReportName(), err)
		}
		if p.Declared[row.ClusterID] == nil {
			p.Declared[row.ClusterID] = map[uint32]any{}
		}
		p.Declared[row.ClusterID][row.AttributeID] = value
	case composition.OriginSemreg:
		if row.Field != "" {
			return nil
		}
		if _, ok := admitted[row.Element]; !ok {
			if row.Mandatory {
				p.Missing = append(p.Missing, row.ReportName())
			}
			return nil
		}
		p.Dynamic = append(p.Dynamic, dynamicRow{Selector{p.Role, row.ClusterID, row.AttributeID}, row})
	default:
		if row.Mandatory {
			p.Missing = append(p.Missing, row.ReportName())
		}
	}
	return nil
}

func missingOf(p *partRows) []string {
	var out []string
	for _, name := range p.Missing {
		if p.Role != "" {
			name = p.Role + ":" + name
		}
		out = append(out, name)
	}
	for _, child := range p.Children {
		out = append(out, missingOf(child)...)
	}
	return out
}

// convert returns the SDK Go value for a Matter type from an encoded or
// declared value. An integer may arrive as a Go integer or as decimal text.
func convert(matterType string, v any) (any, error) {
	if matterType == "bool" {
		switch b := v.(type) {
		case bool:
			return b, nil
		case string:
			parsed, err := strconv.ParseBool(b)
			if err != nil {
				return nil, fmt.Errorf("value %q is not a boolean", b)
			}
			return parsed, nil
		}
		return nil, fmt.Errorf("value %v is not a boolean", v)
	}
	bits, signed, ok := integerKind(matterType)
	if !ok {
		return nil, fmt.Errorf("matter type %q has no value conversion", matterType)
	}
	if signed {
		n, err := toInt(v)
		if err != nil {
			return nil, err
		}
		if bits < 64 && (n < -(1<<(bits-1)) || n >= 1<<(bits-1)) {
			return nil, fmt.Errorf("value %d does not fit %s", n, matterType)
		}
		switch bits {
		case 8:
			return int8(n), nil
		case 16:
			return int16(n), nil
		case 32:
			return int32(n), nil
		default:
			return n, nil
		}
	}
	n, err := toUint(v)
	if err != nil {
		return nil, err
	}
	if bits < 64 && n >= 1<<bits {
		return nil, fmt.Errorf("value %d does not fit %s", n, matterType)
	}
	switch bits {
	case 8:
		return uint8(n), nil
	case 16:
		return uint16(n), nil
	case 32:
		return uint32(n), nil
	default:
		return n, nil
	}
}

func integerKind(matterType string) (bits uint, signed, ok bool) {
	switch matterType {
	case "uint8", "enum8", "bitmap8", "percent":
		return 8, false, true
	case "uint16", "enum16", "bitmap16":
		return 16, false, true
	case "uint32", "bitmap32", "epoch-s":
		return 32, false, true
	case "uint64", "bitmap64":
		return 64, false, true
	case "int8":
		return 8, true, true
	case "int16", "temperature":
		return 16, true, true
	case "int32":
		return 32, true, true
	case "int64", "power-mW", "amperage-mA", "voltage-mV", "energy-mWh":
		return 64, true, true
	}
	return 0, false, false
}

func toInt(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		if n != math.Trunc(n) || math.Abs(n) > 1<<53 {
			return 0, fmt.Errorf("value %v is not an integer", n)
		}
		return int64(n), nil
	case json.Number:
		return strconv.ParseInt(n.String(), 10, 64)
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("value %q is not an integer", n)
		}
		return parsed, nil
	}
	return 0, fmt.Errorf("value %v is not an integer", v)
}

func toUint(v any) (uint64, error) {
	n, err := toInt(v)
	if err != nil {
		if s, ok := v.(string); ok {
			if u, perr := strconv.ParseUint(s, 10, 64); perr == nil {
				return u, nil
			}
		}
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("value %d is negative", n)
	}
	return uint64(n), nil
}
