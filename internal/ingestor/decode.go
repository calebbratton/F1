package ingestor

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// decompress decodes topics suffixed with ".z" (e.g. CarData.z, Position.z).
// F1 sends these as a base64-encoded zlib-compressed JSON string.
func decompress(raw json.RawMessage) (json.RawMessage, error) {
	// The raw value is a JSON string: "\"<base64>\""
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, fmt.Errorf("unmarshal base64 string: %w", err)
	}

	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("zlib reader: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("zlib read: %w", err)
	}

	return json.RawMessage(decompressed), nil
}

// isCompressed returns true for topics that send zlib+base64 payloads.
func isCompressed(topic string) bool {
	return strings.HasSuffix(topic, ".z")
}

// deepMerge merges src into dst recursively.
// F1 sends delta messages with only changed fields; we merge them into
// the last known full state to maintain a current snapshot.
func deepMerge(dst, src map[string]any) map[string]any {
	for k, srcVal := range src {
		dstVal, exists := dst[k]
		if exists {
			srcMap, srcIsMap := toMap(srcVal)
			dstMap, dstIsMap := toMap(dstVal)
			if srcIsMap && dstIsMap {
				dst[k] = deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = srcVal
	}
	return dst
}

// toMap attempts to cast v to map[string]any.
// json.Unmarshal into any produces map[string]any for objects.
func toMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// unmarshalMap decodes a json.RawMessage into map[string]any.
func unmarshalMap(raw json.RawMessage) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}
