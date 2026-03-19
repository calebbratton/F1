package ingestor

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// makeCompressed builds a valid CarData.z / Position.z payload:
// zlib-compress the JSON then base64-encode it, then JSON-encode the string.
func makeCompressed(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, _ := json.Marshal(v)

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(raw)
	w.Close()

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	out, _ := json.Marshal(encoded) // wrap in JSON string
	return json.RawMessage(out)
}

func TestDecompress(t *testing.T) {
	want := map[string]any{"Cars": map[string]any{"1": "VER"}}
	payload := makeCompressed(t, want)

	got, err := decompress(payload)
	if err != nil {
		t.Fatalf("decompress error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	cars, ok := m["Cars"].(map[string]any)
	if !ok || cars["1"] != "VER" {
		t.Errorf("unexpected result: %v", m)
	}
}

func TestDecompress_InvalidBase64(t *testing.T) {
	bad := json.RawMessage(`"not-valid-base64!!!"`)
	_, err := decompress(bad)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDeepMerge_OverwriteScalar(t *testing.T) {
	dst := map[string]any{"gap": "0.000", "pos": 1}
	src := map[string]any{"gap": "+0.432"}
	got := deepMerge(dst, src)
	if got["gap"] != "+0.432" {
		t.Errorf("gap should be updated, got %v", got["gap"])
	}
	if got["pos"] != 1 {
		t.Errorf("pos should be preserved, got %v", got["pos"])
	}
}

func TestDeepMerge_RecursiveObject(t *testing.T) {
	dst := map[string]any{
		"1": map[string]any{"Position": 1.0, "Gap": "0.000"},
	}
	src := map[string]any{
		"1": map[string]any{"Gap": "+0.5"},
	}
	got := deepMerge(dst, src)
	driver := got["1"].(map[string]any)
	if driver["Position"] != 1.0 {
		t.Errorf("Position should be preserved, got %v", driver["Position"])
	}
	if driver["Gap"] != "+0.5" {
		t.Errorf("Gap should be updated, got %v", driver["Gap"])
	}
}

func TestDeepMerge_NewKey(t *testing.T) {
	dst := map[string]any{"a": 1.0}
	src := map[string]any{"b": 2.0}
	got := deepMerge(dst, src)
	if got["a"] != 1.0 || got["b"] != 2.0 {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestIsCompressed(t *testing.T) {
	tests := []struct {
		topic string
		want  bool
	}{
		{"CarData.z", true},
		{"Position.z", true},
		{"TimingData", false},
		{"WeatherData", false},
	}
	for _, tt := range tests {
		if got := isCompressed(tt.topic); got != tt.want {
			t.Errorf("isCompressed(%q) = %v, want %v", tt.topic, got, tt.want)
		}
	}
}
