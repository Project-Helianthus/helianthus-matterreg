package aggregator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/evse"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/partrows"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/clusters/solarpower"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

func TestIdentity(t *testing.T) {
	if DeviceTypeID != 0x000E || DeviceTypeRevision != 2 || BridgedNodeDeviceType != 0x0013 || BridgedNodeRevision != 3 {
		t.Fatal("device type identity")
	}
	if len(Parts()) != 0 || len(Folds()) != 0 || len(MissingRows()) != 0 {
		t.Fatal("the bridge profile has no part, fold or missing row")
	}
}

func TestDescriptorRowIsDeclaredAndNoPartsListIsInvented(t *testing.T) {
	rows := Rows()
	if len(rows) != 1 || rows[0].AttributeID != 0x0000 || rows[0].Origin != composition.OriginDeclared {
		t.Fatalf("rows %#v", rows)
	}
	want := []any{map[string]any{"device_type": "0x000E", "revision": 2}}
	if !reflect.DeepEqual(rows[0].Declared, want) {
		t.Fatalf("declared %v", rows[0].Declared)
	}
}

func TestRequirementsTree(t *testing.T) {
	var top, bridged []string
	var node partrows.Requirement
	for _, r := range Requirements() {
		top = append(top, r.Name+":"+r.Conformance)
		if r.Element == "device_type" {
			node = r
		}
		if r.Conformance != "" && r.Conformance != "M" && r.Mounted {
			t.Fatalf("non-mandatory element mounted %#v", r)
		}
	}
	want := []string{"Descriptor:", "Actions:O", "Identify:O", "Commissioner Control:FabricSynchronization", "Bridged Node:M"}
	if !reflect.DeepEqual(top, want) {
		t.Fatalf("top level %v", top)
	}
	if node.ID != 0x0013 || !node.Mounted {
		t.Fatalf("bridged node %#v", node)
	}
	for _, c := range node.Children {
		bridged = append(bridged, c.Name+":"+c.Conformance)
		if c.Mounted && !c.SDKSupplied {
			t.Fatalf("mounted without the SDK %#v", c)
		}
	}
	wantChildren := []string{
		"Descriptor:", "Bridged Device Basic Information:M", "Power Source Configuration:BridgedPowerSourceInfo, D",
		"Power Source:BridgedPowerSourceInfo", "Administrator Commissioning:FabricSynchronizedNode",
		"Ecosystem Information:FabricSynchronizedNode, O", "Power Source:O",
	}
	if !reflect.DeepEqual(bridged, wantChildren) {
		t.Fatalf("bridged node children %v", bridged)
	}
}

func TestNamedLossesAreDeclaredIdentityLossesOfKindsNoProfileComposes(t *testing.T) {
	covered := map[string]bool{}
	for _, f := range [][]composition.Fold{evse.Folds(), solarpower.Folds()} {
		for _, fold := range f {
			for _, item := range fold.SourceItems {
				covered[item] = true
			}
		}
	}
	composed := map[string]bool{
		"pv.service.inverter": true, "pv.service.system": true, "storage.service.battery": true, "evse.service.evse": true,
		"infrastructure.service.meter": true, "infrastructure.service.grid_connection": true, "infrastructure.service.circuit": true,
		"thermal.service.zone": true, "thermal.service.dhw": true, "thermal.service.heat_generator": true, "thermal.service.pump": true,
		"evse.service.connector": true, "evse.service.phase": true, "pv.service.input": true, "pv.service.phase": true,
		"infrastructure.service.phase": true,
	}
	seen := map[string]bool{}
	losses := NoCounterpart()
	if len(losses) != 36 {
		t.Fatalf("%d losses", len(losses))
	}
	for _, l := range losses {
		if l.Kind != "identity" || !l.Declared || len(l.SourceItems) != 1 || l.SourceItems[0] != l.Name || l.Description == "" {
			t.Fatalf("loss %#v", l)
		}
		if composed[l.Name] || covered[l.Name] || seen[l.Name] || !strings.Contains(l.Name, ".service.") {
			t.Fatalf("loss %s is composed, folded, repeated or not a service", l.Name)
		}
		seen[l.Name] = true
	}
}
