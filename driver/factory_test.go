package driver_test

import (
	"context"
	"encoding/json"
	"flag"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	semdriver "github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"

	"github.com/Project-Helianthus/helianthus-matterreg/driver"
	"github.com/Project-Helianthus/helianthus-matterreg/node"
	"github.com/Project-Helianthus/helianthus-matterreg/node/nodetest"
)

type session struct {
	semdriver.Session
	input semdriver.SemanticInput
}

func (s session) SemanticInput() semdriver.SemanticInput { return s.input }

func options(t *testing.T, store string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(driver.Options{EndpointStore: store})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestBindFlagsRegistersBothOptionsWithTheFalseDefault(t *testing.T) {
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	read := driver.NewFactory().BindFlags(fs)
	enabledFlag := fs.Lookup(driver.FlagEnabled)
	if enabledFlag == nil || enabledFlag.Name != "matter-library-enabled" || enabledFlag.DefValue != "false" {
		t.Fatalf("enabled flag: %+v", enabledFlag)
	}
	storeFlag := fs.Lookup(driver.FlagEndpointStore)
	if storeFlag == nil || storeFlag.Name != "matter-endpoint-store" {
		t.Fatalf("store flag: %+v", storeFlag)
	}
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	enabled, cfg := read()
	if enabled || cfg.Enabled {
		t.Fatal("the library is enabled by default")
	}
	plans, err := driver.NewFactory().Plan(cfg)
	if err != nil || plans != nil {
		t.Fatalf("disabled plan = %v, %v", plans, err)
	}
}

func TestExplicitEnablementAndTheSelectedStoreReachPlanAfterParse(t *testing.T) {
	dir := t.TempDir()
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	factory := driver.NewFactory()
	read := factory.BindFlags(fs)
	if err := fs.Parse([]string{"-matter-library-enabled", "-matter-endpoint-store", dir}); err != nil {
		t.Fatal(err)
	}
	enabled, cfg := read()
	if !enabled || !cfg.Enabled {
		t.Fatal("explicit enablement is not read")
	}
	plans, err := factory.Plan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Driver != driver.Kind || plans[0].Instance != driver.DefaultInstance {
		t.Fatalf("plans = %+v", plans)
	}
	if len(plans[0].Transports) != 1 || plans[0].Transports[0].ConfigKey != dir {
		t.Fatalf("the selected store did not reach the plan: %+v", plans[0].Transports)
	}
}

func TestASecondHostPassesTheSameOptionsWithoutFlags(t *testing.T) {
	dir := t.TempDir()
	cfg := semdriver.Config{Instance: "second", Enabled: true, Options: options(t, dir)}
	plans, err := driver.NewFactory().Plan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Instance != "second" || plans[0].Transports[0].ConfigKey != dir {
		t.Fatalf("plans = %+v", plans)
	}
	instance, err := driver.NewFactory().Construct(context.Background(), plans[0])
	if err != nil || instance == nil {
		t.Fatalf("construct: %v", err)
	}
}

func TestPlanIsPureAndRefusesBadOptions(t *testing.T) {
	factory := driver.NewFactory()
	dir := t.TempDir()
	cfg := semdriver.Config{Instance: "one", Enabled: true, Options: options(t, filepath.Join(dir, "absent")), SecretRef: "ref"}
	first, err := factory.Plan(cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := factory.Plan(cfg)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("plan differs between calls")
	}
	if _, err := os.Stat(filepath.Join(dir, "absent")); !os.IsNotExist(err) {
		t.Fatal("plan touched the filesystem")
	}
	if strings.Contains(string(mustJSON(t, first)), "ref") {
		t.Fatal("plan carries the secret reference")
	}
	for name, raw := range map[string]string{
		"empty store":   `{"endpoint_store":""}`,
		"unknown field": `{"endpoint_store":"x","other":1}`,
		"trailing":      `{"endpoint_store":"x"} {}`,
		"absent":        ``,
	} {
		if _, err := factory.Plan(semdriver.Config{Instance: "one", Enabled: true, Options: json.RawMessage(raw)}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if _, err := factory.Plan(semdriver.Config{Enabled: true, Options: options(t, "x")}); err == nil {
		t.Error("empty instance key planned")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestDescriptorDeclaresNoFamilyAndIsValid(t *testing.T) {
	d := driver.NewFactory().Descriptor()
	if len(d.Families) != 0 || d.Validate() != nil {
		t.Fatalf("descriptor = %+v", d)
	}
}

func TestConstructRefusesAPlanOfAnotherShape(t *testing.T) {
	factory := driver.NewFactory()
	for name, plan := range map[string]semdriver.InstancePlan{
		"other driver": {Driver: "other", Instance: "i"},
		"no transport": {Driver: driver.Kind, Instance: "i"},
	} {
		if _, err := factory.Construct(context.Background(), plan); err == nil {
			t.Errorf("%s constructed", name)
		}
	}
}

func TestStartTakesTheSessionInputAndStopClosesIt(t *testing.T) {
	dir := t.TempDir()
	factory := driver.NewFactory()
	plans, err := factory.Plan(semdriver.Config{Instance: "one", Enabled: true, Options: options(t, dir)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	instance, err := factory.Construct(ctx, plans[0])
	if err != nil {
		t.Fatal(err)
	}
	input := nodetest.NewInput()
	input.Publish(nil, nil)
	if err := instance.Start(ctx, session{input: input}); err != nil {
		t.Fatal(err)
	}
	if input.Captures != 1 {
		t.Fatalf("the session input was captured %d times", input.Captures)
	}
	if _, err := os.Stat(filepath.Join(dir, node.StoreFile)); err != nil {
		t.Fatalf("the store file is not in the selected directory: %v", err)
	}
	if err := instance.Start(ctx, session{input: input}); err == nil {
		t.Fatal("second start accepted")
	}
	instance.FenceWithdrawal()
	if report := instance.Stop(ctx); !report.Confirmed || report.Err != nil {
		t.Fatalf("stop = %+v", report)
	}
	if report := instance.Stop(ctx); !report.Confirmed {
		t.Fatalf("second stop = %+v", report)
	}
}

func TestDriverImportsNeitherTheHostNorTheRuntime(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatal("no source files")
	}
	for _, name := range files {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if strings.Contains(path, "semreg/host") || strings.Contains(path, "semreg/runtime") || strings.Contains(path, "sdkfabric") {
				t.Errorf("%s imports %s", name, path)
			}
		}
	}
}
