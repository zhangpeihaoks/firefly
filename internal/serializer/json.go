package serializer

import (
	"encoding/json"
)

// JSONSerializer implements Serializer for JSON encoding.
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Marshal serializes the value to JSON bytes.
func (s *JSONSerializer) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal deserializes JSON bytes into the value.
func (s *JSONSerializer) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// ContentType returns the JSON content type.
func (s *JSONSerializer) ContentType() string {
	return "application/json"
}
