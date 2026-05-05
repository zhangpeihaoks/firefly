// Package serializer provides serialization abstractions for the Firefly framework.
// It defines the Serializer interface and provides JSON and Protobuf implementations.
package serializer

// SerializerMode represents the serialization mode (JSON, Protobuf, etc.)
type SerializerMode string

const (
	// ModeJSON represents JSON serialization mode
	ModeJSON SerializerMode = "json"
	// ModeProtobuf represents Protobuf serialization mode
	ModeProtobuf SerializerMode = "protobuf"
)

// Serializer is the serialization interface.
// It provides methods for marshaling and unmarshaling data.
type Serializer interface {
	// Marshal serializes the given value to bytes.
	Marshal(v any) ([]byte, error)
	// Unmarshal deserializes bytes into the given value.
	Unmarshal(data []byte, v any) error
	// ContentType returns the Content-Type for this serializer.
	ContentType() string
}
