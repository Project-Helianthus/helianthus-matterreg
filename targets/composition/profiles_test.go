package composition

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Project-Helianthus/helianthus-matterreg/targets/pilot"
)

func TestLoadAcceptedPilot(t *testing.T) {
	source, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal("pilot kind is not loaded")
	}
	if _, ok := source.Lookup("resource.other"); ok {
		t.Fatal("unknown kind fell back to a profile")
	}
	onDisk, err := os.ReadFile(filepath.Join("..", "pilot", "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(profile.Document, bytes.TrimSpace(onDisk)) || !bytes.Equal(profile.Document, bytes.TrimSpace(pilot.Document())) {
		t.Fatal("loaded document is not the pilot entry")
	}
	mandatory := 0
	for _, row := range profile.Attributes {
		if row.Mandatory {
			mandatory++
		}
	}
	if mandatory == 0 {
		t.Fatal("pilot mandatory rows are empty")
	}
	if len(profile.Commands) != 0 || len(profile.Parts) != 0 || profile.PartKey != "" {
		t.Fatalf("pilot commands %#v parts %#v", profile.Commands, profile.Parts)
	}
	profile.Commands = append(profile.Commands, CommandRow{ClusterID: 1, ElementID: 2, Operation: "mutated"})
	again, ok := source.Lookup(pilot.ResourceKind)
	if !ok || len(again.Commands) != 0 {
		t.Fatal("lookup shares command storage")
	}
	if _, err := sourceFrom([]Profile{profile, profile}); err == nil {
		t.Fatal("duplicate resource kind accepted")
	}
}

func TestProfileStructure(t *testing.T) {
	source, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	base, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal(err)
	}
	cases := map[string]func(*Profile){
		"duplicate part role": func(p *Profile) {
			p.Parts = []PartSpec{{Role: "sensor", DeviceTypeID: 0x0510, Count: 1}, {Role: "sensor", DeviceTypeID: 0x0510, Count: 1}}
		},
		"empty part role": func(p *Profile) {
			p.Parts = []PartSpec{{Role: "", DeviceTypeID: 0x0510, Count: 1}}
		},
		"wrong part selector": func(p *Profile) {
			p.Commands = []CommandRow{{PartKey: "missing", ClusterID: 0x0090, ElementID: 0x0005}}
		},
		"duplicate command selector": func(p *Profile) {
			p.Parts = []PartSpec{{Role: "sensor", DeviceTypeID: 0x0510, Count: 1}}
			p.Commands = []CommandRow{
				{PartKey: "sensor", ClusterID: 0x0090, ElementID: 0x0005, Operation: "semreg.operation.one", Capability: "semreg.capability.one", InverseEncoder: "encoder.one"},
				{PartKey: "sensor", ClusterID: 0x0090, ElementID: 0x0005, Operation: "semreg.operation.two", Capability: "semreg.capability.two", InverseEncoder: "encoder.two"},
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			profile := base
			profile.Parts = nil
			profile.Commands = nil
			mutate(&profile)
			if err := validateProfile(profile); err == nil {
				t.Fatalf("accepted %s", name)
			}
		})
	}
	if err := validateProfile(base); err != nil {
		t.Fatal(err)
	}
}

func TestPilotKindResolvesOnceWithComposedIdentity(t *testing.T) {
	raw, err := os.ReadFile("profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fileDoc struct {
		SchemaVersion int              `json:"schema_version"`
		Profiles      []map[string]any `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &fileDoc); err != nil {
		t.Fatal(err)
	}
	if fileDoc.SchemaVersion != 1 || len(fileDoc.Profiles) != 1 {
		t.Fatalf("composition file %d entries", len(fileDoc.Profiles))
	}
	lockRaw, err := os.ReadFile(filepath.Join("..", "..", "sdk", "matter-sdk.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		SDK     map[string]any `json:"sdk"`
		Mapping map[string]any `json:"mapping"`
	}
	if err := json.Unmarshal(lockRaw, &lock); err != nil {
		t.Fatal(err)
	}
	entry := fileDoc.Profiles[0]
	for _, field := range []string{"module", "release_tag", "checksum"} {
		if entry[field] != lock.SDK[field] {
			t.Fatalf("entry %s %v differs from the lock", field, entry[field])
		}
	}
	if !reflect.DeepEqual(entry["mapping"], lock.Mapping) {
		t.Fatal("entry mapping differs from the lock")
	}
	pilotFile, err := os.ReadFile(filepath.Join("..", "pilot", "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pilotEntry map[string]any
	if err := json.Unmarshal(pilotFile, &pilotEntry); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(entry, pilotEntry) {
		t.Fatal("composition entry differs in content from the pilot profile")
	}
	source, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(source.byKind) != 1 {
		t.Fatalf("loaded %d kinds", len(source.byKind))
	}
	profile, ok := source.Lookup(pilot.ResourceKind)
	if !ok {
		t.Fatal("pilot kind is not loaded")
	}
	var loaded map[string]any
	if err := json.Unmarshal(profile.Document, &loaded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, entry) {
		t.Fatal("pilot kind does not resolve to the composition entry")
	}
}
