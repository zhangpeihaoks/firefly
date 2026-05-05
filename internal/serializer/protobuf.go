package serializer

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

// ProtobufSerializer implements Serializer for Protobuf encoding.
type ProtobufSerializer struct{}

// NewProtobufSerializer creates a new Protobuf serializer.
func NewProtobufSerializer() *ProtobufSerializer {
	return &ProtobufSerializer{}
}

// Marshal serializes the value to Protobuf bytes.
// The value must implement proto.Message interface.
func (s *ProtobufSerializer) Marshal(v any) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, errors.New("serializer: value is not a proto.Message")
	}
	return proto.Marshal(msg)
}

// Unmarshal deserializes Protobuf bytes into the value.
// The value must be a pointer to a proto.Message.
func (s *ProtobufSerializer) Unmarshal(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errors.New("serializer: value is not a proto.Message")
	}
	return proto.Unmarshal(data, msg)
}

// ContentType returns the Protobuf content type.
func (s *ProtobufSerializer) ContentType() string {
	return "application/x-protobuf"
}
