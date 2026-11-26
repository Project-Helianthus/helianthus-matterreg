// Package nodetest holds the synthetic profiles, roster rows and the in-process
// site launcher the bridge node tests and the qualification harnesses share.
// Nothing here is a product profile.
package nodetest

import (
	"context"
	"fmt"
	"path/filepath"

	"thingfabric.dev/go-fabric/fabrictest"
	gfnode "thingfabric.dev/go-fabric/node"

	bridgenode "github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

// Synthetic resource kinds and catalog elements.
const (
	KindZone       = "resource.synthetic.zone"
	KindAssembly   = "resource.synthetic.assembly"
	KindIncomplete = "resource.synthetic.incomplete"
	KindUnmapped   = "resource.synthetic.unmapped"

	ElementTemperature = "synthetic.zone.temperature"
	ElementTolerance   = "synthetic.zone.tolerance"
	ElementPartValue   = "synthetic.assembly.temperature"
	ElementOrphan      = "synthetic.incomplete.value"
	ElementUnrostered  = "synthetic.unrostered.value"

	ElementBackupValue = "synthetic.assembly.backup_temperature"

	PartSensor = "sensor"
	PartBackup = "backup"

	ClusterTemperature uint32 = 0x0402
	AttrMeasured       uint32 = 0x0000
	AttrMin            uint32 = 0x0001
	AttrMax            uint32 = 0x0002
	AttrTolerance      uint32 = 0x0003
)

func descriptorRows(deviceType string, servers ...any) []composition.AttributeRow {
	return []composition.AttributeRow{
		{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0000, Name: "DeviceTypeList", MatterType: "list", Mandatory: true, Origin: composition.OriginDeclared, Declared: []any{map[string]any{"device_type": deviceType, "revision": 1}}},
		{ClusterID: 0x001D, ClusterName: "Descriptor", AttributeID: 0x0001, Name: "ServerList", MatterType: "list", Mandatory: true, Origin: composition.OriginDeclared, Declared: servers},
	}
}

func temperatureRows(measured, tolerance string) []composition.AttributeRow {
	rows := []composition.AttributeRow{
		{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMeasured, Name: "MeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginSemreg, Element: measured, Dimension: "synthetic.dimension", Encoder: "kelvin_to_centicelsius"},
		{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMin, Name: "MinMeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginDeclared, Declared: -5000},
		{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMax, Name: "MaxMeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginDeclared, Declared: 15000},
	}
	if tolerance != "" {
		rows = append(rows, composition.AttributeRow{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrTolerance, Name: "Tolerance", MatterType: "uint16", Origin: composition.OriginSemreg, Element: tolerance, Dimension: "synthetic.dimension"})
	}
	return rows
}

// Profiles returns the synthetic profiles, complete in their mandatory elements
// except KindIncomplete, which names a mandatory row that has no source.
func Profiles() []composition.Profile {
	zone := composition.Profile{
		Kind:       KindZone,
		Attributes: append(descriptorRows("0x0302", "0x001D", "0x0402"), temperatureRows(ElementTemperature, ElementTolerance)...),
	}
	assembly := composition.Profile{
		Kind:       KindAssembly,
		Attributes: descriptorRows("0x0013", "0x001D"),
		Parts: []composition.PartSpec{{
			Role: PartSensor, DeviceTypeID: 0x0302, Revision: 2, Count: 1,
			Servers:    []composition.ServerRow{{ClusterID: ClusterTemperature, Name: "Temperature Measurement"}},
			Attributes: temperatureRows(ElementPartValue, ""),
		}, {
			Role: PartBackup, DeviceTypeID: 0x0302, Revision: 2, Count: 1,
			Servers:    []composition.ServerRow{{ClusterID: ClusterTemperature, Name: "Temperature Measurement"}},
			Attributes: temperatureRows(ElementBackupValue, ""),
		}},
	}
	incomplete := composition.Profile{
		Kind: KindIncomplete,
		Attributes: append(descriptorRows("0x0302", "0x001D", "0x0402"), []composition.AttributeRow{
			{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMeasured, Name: "MeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginSemreg, Element: ElementOrphan},
			{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMin, Name: "MinMeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginMissing, Detail: "no source"},
			{ClusterID: ClusterTemperature, ClusterName: "Temperature Measurement", AttributeID: AttrMax, Name: "MaxMeasuredValue", MatterType: "int16", Nullable: true, Mandatory: true, Origin: composition.OriginDeclared, Declared: 15000},
		}...),
	}
	unmapped := composition.Profile{
		Kind:       KindUnmapped,
		Attributes: append(descriptorRows("0x0302", "0x001D", "0x0402"), temperatureRows(ElementUnrostered, "")...),
	}
	return []composition.Profile{zone, assembly, incomplete, unmapped}
}

// Roster returns the synthetic roster rows. ElementUnrostered and
// ElementTolerance have no row.
func Roster() []publication.Row {
	rows := []publication.Row{}
	for _, element := range []string{ElementTemperature, ElementPartValue, ElementBackupValue, ElementOrphan} {
		rows = append(rows, publication.Row{Element: element, Kind: "attribute", Disposition: publication.Transformed})
	}
	return rows
}

// Config returns a node configuration over dir with the synthetic profiles.
func Config(dir string) bridgenode.Config {
	return bridgenode.Config{
		StoreDir: dir,
		Profiles: Profiles(),
		Roster:   Roster(),
		ProductDescription: gfnode.ProductDescription{
			VendorID: bridgenode.DevelopmentVendorID, ProductID: bridgenode.ProductID,
		},
		Commissioning: gfnode.Commissioning{
			Passcode: fabrictest.DefaultPasscode, Discriminator: fabrictest.DefaultDiscriminator,
			PBKDFIterations: fabrictest.DefaultPBKDFIterations, Salt: []byte("SPAKE2P Key Salt"),
		},
		BasicInformation: gfnode.BasicInformation{
			VendorName: "Synthetic", ProductName: "Synthetic bridge", NodeLabel: "bridge", SerialNumber: "SYN-1",
			HardwareVersion: 1, HardwareVersionString: "1", SoftwareVersion: 1, SoftwareVersionString: "1",
		},
	}
}

// StorePath is the SDK store file inside dir.
func StorePath(dir string) string { return filepath.Join(dir, bridgenode.StoreFile) }

// SiteLauncher starts every bridge node as a device of one fabrictest site.
type SiteLauncher struct {
	Site    *fabrictest.Site
	Devices []*fabrictest.Device
}

type siteRunning struct{ d *fabrictest.Device }

// Launch implements the bridge's Launcher. After a stop, the next launch starts
// the same device again on its store, which keeps its address.
func (l *SiteLauncher) Launch(ctx context.Context, cfg gfnode.Config) (bridgenode.Running, error) {
	if n := len(l.Devices); n > 0 && !l.Devices[n-1].Running() {
		if err := l.Devices[n-1].Start(ctx); err != nil {
			return nil, err
		}
		return siteRunning{l.Devices[n-1]}, nil
	}
	d := l.Site.AddNode(cfg)
	l.Devices = append(l.Devices, d)
	return siteRunning{d}, nil
}

func (r siteRunning) Node() *gfnode.Node { return r.d.Node() }

func (r siteRunning) Stop(ctx context.Context) error { return r.d.Stop(ctx) }

// Last returns the device of the latest launch.
func (l *SiteLauncher) Last() (*fabrictest.Device, error) {
	if len(l.Devices) == 0 {
		return nil, fmt.Errorf("nodetest: nothing launched")
	}
	return l.Devices[len(l.Devices)-1], nil
}
