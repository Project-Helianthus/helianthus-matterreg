package node

import (
	"fmt"

	"thingfabric.dev/go-fabric/cluster/spec"
	"thingfabric.dev/go-fabric/device"
	"thingfabric.dev/go-fabric/endpoint"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// partKeySeparator cannot occur in a resource id: a resource id never holds a
// line break.
const partKeySeparator = "\npart\n"

// RootKey is the endpoint stable key of a resource: its admitted resource id.
func RootKey(resourceID string) device.SourceKey { return device.StringKey(resourceID) }

// PartKey is the stable key of a protocol part: the exact resource id and the
// typed profile role of the part.
func PartKey(resourceID, role string) device.SourceKey {
	return device.StringKey(resourceID + partKeySeparator + role)
}

// Mount is what composeEndpoint hands the aggregator for one resource: the
// stable resource id, the servers and the profile command rows of the mount
// point. A command layer installs its handlers on the servers before the
// endpoint is added.
type Mount struct {
	Ref      string
	Commands map[Selector][]composition.CommandRow
	Servers  map[string]map[uint32]spec.AttributeSetter
	root     *device.Device
}

// Spec implements the SDK's node.Endpoint.
func (m *Mount) Spec() endpoint.Spec { return m.root.Spec() }

// composeEndpoint builds the endpoint for one resource from the classified
// rows of its profile: the root and, recursively, the typed parts.
func composeEndpoint(resourceID string, rows *partRows, initial map[Selector]Value, commands map[Selector][]composition.CommandRow) (*Mount, error) {
	mount := &Mount{Ref: resourceID, Commands: commands, Servers: map[string]map[uint32]spec.AttributeSetter{}}
	root, err := composePart(resourceID, rows, initial, mount)
	if err != nil {
		return nil, err
	}
	mount.root = root
	return mount, nil
}

func composePart(resourceID string, rows *partRows, initial map[Selector]Value, mount *Mount) (*device.Device, error) {
	deviceType, ok := device.Lookup(rows.DeviceTypeID)
	if !ok {
		return nil, fmt.Errorf("device type 0x%04X is not in the SDK", rows.DeviceTypeID)
	}
	key := RootKey(resourceID)
	if rows.Role != "" {
		key = PartKey(resourceID, rows.Role)
	}
	declarations, err := declarations(rows, initial)
	if err != nil {
		return nil, err
	}
	built, err := device.New(deviceType, device.Options{
		StableKey: key, Name: resourceID, DeviceAddress: resourceID,
	}, declarations...)
	if err != nil {
		return nil, fmt.Errorf("device type 0x%04X: %w", rows.DeviceTypeID, err)
	}
	servers := map[uint32]spec.AttributeSetter{}
	for _, id := range rows.Servers {
		if setter, ok := built.Server(id).(spec.AttributeSetter); ok {
			servers[id] = setter
		}
	}
	mount.Servers[rows.Role] = servers
	for _, child := range rows.Children {
		part, err := composePart(resourceID, child, initial, mount)
		if err != nil {
			return nil, err
		}
		if err := built.AddPart(part); err != nil {
			return nil, err
		}
	}
	return built, nil
}

type clusterDeclaration struct{ decl spec.ClusterDeclaration }

func (c clusterDeclaration) ClusterID() uint32 { return c.decl.Cluster }

func (c clusterDeclaration) ClusterDeclaration() spec.ClusterDeclaration { return c.decl }

func declarations(rows *partRows, initial map[Selector]Value) ([]device.Declaration, error) {
	byCluster := map[uint32]map[uint32]any{}
	for cluster, attributes := range rows.Declared {
		byCluster[cluster] = map[uint32]any{}
		for id, value := range attributes {
			byCluster[cluster][id] = value
		}
	}
	for _, row := range rows.Dynamic {
		value, ok := initial[row.Selector]
		if !ok {
			if row.Row.Nullable {
				value = Value{Null: true}
			} else {
				continue
			}
		}
		stored, err := stored(row.Row.MatterType, row.Row.Nullable, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", row.Row.ReportName(), err)
		}
		if byCluster[row.ClusterID] == nil {
			byCluster[row.ClusterID] = map[uint32]any{}
		}
		byCluster[row.ClusterID][row.AttributeID] = stored
	}
	var out []device.Declaration
	for _, id := range rows.Servers {
		decl := spec.ClusterDeclaration{Cluster: id, Initial: byCluster[id]}
		if rows.HasFeatures[id] {
			decl.Features, decl.HasFeatures = rows.Features[id], true
		}
		out = append(out, clusterDeclaration{decl})
	}
	return out, nil
}

func stored(matterType string, nullable bool, value Value) (any, error) {
	if value.Null {
		if !nullable {
			return nil, fmt.Errorf("attribute is not nullable")
		}
		return nil, nil
	}
	return convert(matterType, value.Value)
}
