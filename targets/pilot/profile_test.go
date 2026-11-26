package pilot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDocumentIsTheAdmissionObject(t *testing.T) {
	raw := Document()
	onDisk, err := os.ReadFile("profile.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, onDisk) {
		t.Fatal("embedded profile differs from profile.json")
	}
	decoded := decodeObject(t, raw)
	wantKeys := []string{"checksum", "clusters", "device_type", "electrical_power_measurement", "mapping", "module", "profile_id", "release_tag", "schema_version", "status"}
	if !reflect.DeepEqual(sortedKeys(decoded), wantKeys) {
		t.Fatalf("keys %v", sortedKeys(decoded))
	}
	if decoded["schema_version"] != json.Number("1") || decoded["status"] != "accepted" || decoded["profile_id"] == "" {
		t.Fatalf("identity %#v", decoded["profile_id"])
	}
	lock := decodeObject(t, mustRead(t, filepath.Join("..", "..", "sdk", "matter-sdk.lock.json")))
	sdk := lock["sdk"].(map[string]any)
	for _, field := range []string{"module", "release_tag", "checksum"} {
		if decoded[field] != sdk[field] {
			t.Fatalf("%s %v", field, decoded[field])
		}
	}
	if !reflect.DeepEqual(decoded["mapping"], lock["mapping"]) || !reflect.DeepEqual(decoded["device_type"], lock["target"].(map[string]any)["device_type"]) {
		t.Fatal("profile pins differ from the lock")
	}
	pin := lock["accepted_runtime_composition_profile_sha256"]
	if digest(canonicalJSON(t, raw)) == pin {
		t.Fatal("the pilot profile alone matches the lock pin")
	}
	if digest(canonicalJSON(t, mustRead(t, filepath.Join("..", "composition", "profiles.json")))) != pin {
		t.Fatal("composition file digest differs from the lock pin")
	}
	clusters := decoded["clusters"].([]any)
	found := map[string]string{}
	for _, item := range clusters {
		cluster := item.(map[string]any)
		found[cluster["id"].(string)] = cluster["name"].(string)
	}
	if found["0x001D"] != "Descriptor" || found["0x009C"] != "Power Topology" || found["0x0090"] != "Electrical Power Measurement" {
		t.Fatalf("clusters %#v", found)
	}
	epm := decoded["electrical_power_measurement"].(map[string]any)
	if !reflect.DeepEqual(sortedKeys(epm), []string{"accuracy", "active_power", "number_of_measurement_types", "power_mode"}) {
		t.Fatalf("epm keys %v", sortedKeys(epm))
	}
}

func TestAttributeRowsNameEveryMandatorySource(t *testing.T) {
	rows := Rows()
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	again := Rows()
	again[0].Declared.([]any)[0].(map[string]any)["revision"] = 9
	if Rows()[0].Declared.([]any)[0].(map[string]any)["revision"] == 9 {
		t.Fatal("row copy shares declared values")
	}
	var semreg, declared, missing int
	names := map[string]AttributeRow{}
	for _, row := range rows {
		if !row.Mandatory && row.Origin != OriginSemreg {
			t.Fatalf("optional row %s has no semreg source", row.ReportName())
		}
		names[row.ReportName()] = row
		switch row.Origin {
		case OriginSemreg:
			semreg++
			if row.Element == "" || row.Encoder == "" || row.PackID == "" || row.PackVersion == "" || row.Dimension == "" {
				t.Fatalf("semreg row %s is incomplete", row.ReportName())
			}
		case OriginDeclared:
			declared++
			if row.Declared == nil {
				t.Fatalf("declared row %s has no value", row.ReportName())
			}
		case OriginMissing:
			missing++
			if row.Element != "" || row.Declared != nil {
				t.Fatalf("missing row %s invents a source", row.ReportName())
			}
		default:
			t.Fatalf("row %s origin %s", row.ReportName(), row.Origin)
		}
	}
	if semreg != 1 || declared == 0 || missing == 0 {
		t.Fatalf("semreg %d declared %d missing %d", semreg, declared, missing)
	}
	current := names["Electrical Power Measurement.ActiveCurrent"]
	if current.AttributeID != 0x0005 || current.Element != "evse.ac.current" || current.Encoder != "ampere_to_milliampere" || !current.Nullable || current.Mandatory {
		t.Fatalf("current row %#v", current)
	}
	device := names["Descriptor.DeviceTypeList"].Declared.([]any)[0].(map[string]any)
	if device["device_type"] != "0x0510" || device["revision"] != 1 {
		t.Fatalf("device type %#v", device)
	}
	for _, name := range []string{
		"Power Topology.FeatureMap",
		"Electrical Power Measurement.FeatureMap",
		"Electrical Power Measurement.PowerMode",
		"Electrical Power Measurement.NumberOfMeasurementTypes",
		"Electrical Power Measurement.Accuracy",
		"Electrical Power Measurement.Accuracy.MeasurementType",
		"Electrical Power Measurement.Accuracy.Measured",
		"Electrical Power Measurement.Accuracy.MinMeasuredValue",
		"Electrical Power Measurement.Accuracy.MaxMeasuredValue",
		"Electrical Power Measurement.Accuracy.AccuracyRanges",
		"Electrical Power Measurement.Accuracy.RangeMin",
		"Electrical Power Measurement.Accuracy.RangeMax",
		"Electrical Power Measurement.ActivePower",
	} {
		row, ok := names[name]
		if !ok || row.Origin != OriginMissing || !row.Mandatory {
			t.Fatalf("missing row %s = %#v", name, row)
		}
	}
	if len(Folds()) != 1 || !Folds()[0].Declared || Folds()[0].Kind != "identity" || Folds()[0].SourceItems[0] != "evse.ac.current" {
		t.Fatalf("folds %#v", Folds())
	}
}

func decodeObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%T", value)
	}
	return object
}

func canonicalJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(decodeObject(t, raw)); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmtHex(sum[:])
}

func fmtHex(raw []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(raw)*2)
	for i, b := range raw {
		out[i*2] = digits[b>>4]
		out[i*2+1] = digits[b&0x0f]
	}
	return string(out)
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
