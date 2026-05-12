// Package serializer provides unit tests for Protobuf serializer.
package serializer

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestProtobufSerializerMarshal tests the Marshal method of ProtobufSerializer
// Validates: Requirements - 序列化需求 (Property 45: Protobuf 序列化正确性)
func TestProtobufSerializerMarshal(t *testing.T) {
	serializer := NewProtobufSerializer()

	tests := []struct {
		name    string
		input   any
		wantErr bool
		// checkEmpty: if true, allow empty bytes as valid (for zero-value messages)
		checkEmpty bool
	}{
		{
			name:       "timestamp message",
			input:      timestamppb.Now(),
			wantErr:    false,
			checkEmpty: false,
		},
		{
			name:       "duration message",
			input:      durationpb.New(100),
			wantErr:    false,
			checkEmpty: false,
		},
		{
			name:       "empty message",
			input:      &emptypb.Empty{},
			wantErr:    false,
			checkEmpty: true, // Empty message serializes to empty bytes (valid)
		},
		{
			name:    "string wrapper",
			input:   wrapperspb.String("hello"),
			wantErr: false,
		},
		{
			name:    "int32 wrapper",
			input:   wrapperspb.Int32(42),
			wantErr: false,
		},
		{
			name:    "bool wrapper true",
			input:   wrapperspb.Bool(true),
			wantErr: false,
		},
		{
			name:       "bool wrapper false",
			input:      wrapperspb.Bool(false),
			wantErr:    false,
			checkEmpty: true, // false/zero values serialize to empty bytes (valid)
		},
		{
			name:    "bytes wrapper",
			input:   wrapperspb.Bytes([]byte("test")),
			wantErr: false,
		},
		{
			name:    "struct message",
			input:   &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "test"}},
			wantErr: false,
		},
		{
			name:       "any message",
			input:      &anypb.Any{},
			wantErr:    false,
			checkEmpty: true, // Empty Any message serializes to empty bytes (valid)
		},
		{
			name:       "zero duration",
			input:      durationpb.New(0),
			wantErr:    false,
			checkEmpty: true,
		},
		{
			name:       "zero string wrapper",
			input:      wrapperspb.String(""),
			wantErr:    false,
			checkEmpty: true,
		},
		{
			name:       "zero int32 wrapper",
			input:      wrapperspb.Int32(0),
			wantErr:    false,
			checkEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serializer.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.checkEmpty && len(got) == 0 {
				t.Error("Marshal() returned empty bytes for valid message")
			}
		})
	}
}

// TestProtobufSerializerMarshalError tests error cases for Marshal
func TestProtobufSerializerMarshalError(t *testing.T) {
	serializer := NewProtobufSerializer()

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "non-proto.Message string",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "non-proto.Message int",
			input:   42,
			wantErr: true,
		},
		{
			name:    "non-proto.Message struct",
			input:   struct{ Name string }{Name: "test"},
			wantErr: true,
		},
		{
			name:    "non-proto.Message map",
			input:   map[string]string{"key": "value"},
			wantErr: true,
		},
		{
			name:    "non-proto.Message slice",
			input:   []int{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "nil",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "pointer to proto.Message",
			input:   timestamppb.Now(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := serializer.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestProtobufSerializerUnmarshal tests the Unmarshal method of ProtobufSerializer
// Validates: Requirements - 序列化需求 (Property 45: Protobuf 序列化正确性)
func TestProtobufSerializerUnmarshal(t *testing.T) {
	serializer := NewProtobufSerializer()

	// First marshal some data to use in unmarshal tests
	marshaledData, err := serializer.Marshal(timestamppb.Now())
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	tests := []struct {
		name       string
		data       []byte
		target     any
		wantErr    bool
		allowEmpty bool // Allow empty data as valid input
	}{
		{
			name:       "timestamp unmarshal",
			data:       marshaledData,
			target:     &timestamppb.Timestamp{},
			wantErr:    false,
			allowEmpty: false,
		},
		{
			name:       "unmarshal into empty data",
			data:       []byte{},
			target:     &timestamppb.Timestamp{},
			wantErr:    false, // Empty bytes is valid for protobuf (zero value)
			allowEmpty: true,
		},
		{
			name:       "unmarshal nil data",
			data:       nil,
			target:     &timestamppb.Timestamp{},
			wantErr:    false, // nil data is treated as valid empty input
			allowEmpty: true,
		},
		{
			name:       "duration unmarshal",
			data:       marshaledData,
			target:     &durationpb.Duration{},
			wantErr:    false, // Will produce a valid zero-value duration
			allowEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := serializer.Unmarshal(tt.data, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestProtobufSerializerUnmarshalError tests error cases for Unmarshal
func TestProtobufSerializerUnmarshalError(t *testing.T) {
	serializer := NewProtobufSerializer()

	// Valid marshaled data
	validData, _ := serializer.Marshal(timestamppb.Now())

	tests := []struct {
		name    string
		data    []byte
		target  any
		wantErr bool
	}{
		{
			name:    "non-proto.Message target string",
			data:    validData,
			target:  "not a proto message",
			wantErr: true,
		},
		{
			name:    "non-proto.Message target struct",
			data:    validData,
			target:  struct{ Name string }{},
			wantErr: true,
		},
		{
			name:    "non-proto.Message target int",
			data:    validData,
			target:  42,
			wantErr: true,
		},
		{
			name:    "nil target",
			data:    validData,
			target:  nil,
			wantErr: true,
		},
		{
			name:    "invalid protobuf data",
			data:    []byte{0xff, 0xfe, 0xfd},
			target:  &timestamppb.Timestamp{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := serializer.Unmarshal(tt.data, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestProtobufSerializerContentType tests the ContentType method
// Validates: Requirements - 序列化需求 (Property 47: 序列化 Content-Type)
func TestProtobufSerializerContentType(t *testing.T) {
	serializer := NewProtobufSerializer()

	tests := []struct {
		name string
		want string
	}{
		{
			name: "protobuf content type",
			want: "application/protobuf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serializer.ContentType()
			if got != tt.want {
				t.Errorf("ContentType() = %v, want %v", got, tt.want)
			}
		})
	}

	// Additional test: multiple calls return consistent result
	for i := 0; i < 10; i++ {
		if serializer.ContentType() != "application/protobuf" {
			t.Errorf("ContentType() not consistent on call %d", i)
		}
	}
}

// TestProtobufSerializerRoundTripTimestamp tests round trip for timestamp
func TestProtobufSerializerRoundTripTimestamp(t *testing.T) {
	serializer := NewProtobufSerializer()

	original := timestamppb.Now()

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &timestamppb.Timestamp{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare - seconds should match
	if result.Seconds != original.Seconds {
		t.Errorf("RoundTrip: got seconds %v, want %v", result.Seconds, original.Seconds)
	}
}

// TestProtobufSerializerRoundTripDuration tests round trip for duration
func TestProtobufSerializerRoundTripDuration(t *testing.T) {
	serializer := NewProtobufSerializer()

	original := durationpb.New(100)

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &durationpb.Duration{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if result.Seconds != original.Seconds {
		t.Errorf("RoundTrip: got seconds %v, want %v", result.Seconds, original.Seconds)
	}
}

// TestProtobufSerializerRoundTripStringWrapper tests round trip for string wrapper
func TestProtobufSerializerRoundTripStringWrapper(t *testing.T) {
	serializer := NewProtobufSerializer()

	original := wrapperspb.String("hello world")

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &wrapperspb.StringValue{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if result.Value != original.Value {
		t.Errorf("RoundTrip: got %v, want %v", result.Value, original.Value)
	}
}

// TestProtobufSerializerRoundTripIntWrapper tests round trip for int32 wrapper
func TestProtobufSerializerRoundTripIntWrapper(t *testing.T) {
	serializer := NewProtobufSerializer()

	original := wrapperspb.Int32(42)

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &wrapperspb.Int32Value{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if result.Value != original.Value {
		t.Errorf("RoundTrip: got %v, want %v", result.Value, original.Value)
	}
}

// TestProtobufSerializerRoundTripBoolWrapper tests round trip for bool wrapper
func TestProtobufSerializerRoundTripBoolWrapper(t *testing.T) {
	serializer := NewProtobufSerializer()

	tests := []struct {
		name     string
		original *wrapperspb.BoolValue
	}{
		{
			name:     "true",
			original: wrapperspb.Bool(true),
		},
		{
			name:     "false",
			original: wrapperspb.Bool(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := serializer.Marshal(tt.original)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			result := &wrapperspb.BoolValue{}
			err = serializer.Unmarshal(data, result)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if result.Value != tt.original.Value {
				t.Errorf("RoundTrip: got %v, want %v", result.Value, tt.original.Value)
			}
		})
	}
}

// TestProtobufSerializerRoundTripStruct tests round trip for structpb.Value
func TestProtobufSerializerRoundTripStruct(t *testing.T) {
	serializer := NewProtobufSerializer()

	original, _ := structpb.NewValue(map[string]interface{}{
		"name": "test",
		"age":  30,
	})

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &structpb.Value{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify the struct content
	structResult, ok := result.Kind.(*structpb.Value_StructValue)
	if !ok {
		t.Fatal("Unmarshal result is not a struct")
	}
	if structResult.StructValue.Fields["name"].GetStringValue() != "test" {
		t.Errorf("Name field = %v, want test", structResult.StructValue.Fields["name"].GetStringValue())
	}
}

// TestProtobufSerializerInterface tests that ProtobufSerializer implements Serializer interface
func TestProtobufSerializerInterface(t *testing.T) {
	// Compile-time interface check
	var _ Serializer = NewProtobufSerializer()
}

// TestProtobufSerializerNew tests the NewProtobufSerializer constructor
func TestProtobufSerializerNew(t *testing.T) {
	serializer := NewProtobufSerializer()
	if serializer == nil {
		t.Error("NewProtobufSerializer() returned nil")
	}

	// Test that multiple calls return usable instances
	s1 := NewProtobufSerializer()
	s2 := NewProtobufSerializer()
	if s1 == nil || s2 == nil {
		t.Error("NewProtobufSerializer() returned nil instance")
	}

	// Both should work independently
	data1, _ := s1.Marshal(timestamppb.Now())
	data2, _ := s2.Marshal(wrapperspb.String("test"))

	if bytes.Equal(data1, data2) {
		t.Error("Serializers should produce different outputs for different inputs")
	}
}

// TestProtobufSerializerEmptyMessage tests round trip for empty message
func TestProtobufSerializerEmptyMessage(t *testing.T) {
	serializer := NewProtobufSerializer()

	original := &emptypb.Empty{}

	// Marshal
	data, err := serializer.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal
	result := &emptypb.Empty{}
	err = serializer.Unmarshal(data, result)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
}

// TestProtobufSerializerMultipleMessages tests serializing different message types
func TestProtobufSerializerMultipleMessages(t *testing.T) {
	serializer := NewProtobufSerializer()

	messages := []struct {
		msg       any
		allowZero bool // Allow zero-value messages that serialize to empty bytes
	}{
		{timestamppb.Now(), false},
		{durationpb.New(0), true},          // Zero duration serializes to empty bytes (valid)
		{&emptypb.Empty{}, true},           // Empty message serializes to empty bytes (valid)
		{wrapperspb.String(""), true},      // Empty string serializes to empty bytes (valid)
		{wrapperspb.Int32(0), true},        // Zero int32 serializes to empty bytes (valid)
		{wrapperspb.Bool(false), true},     // False bool serializes to empty bytes (valid)
		{wrapperspb.Bytes([]byte{}), true}, // Empty bytes serialize to empty bytes (valid)
	}

	for _, tc := range messages {
		data, err := serializer.Marshal(tc.msg)
		if err != nil {
			t.Errorf("Marshal() error for %T: %v", tc.msg, err)
			continue
		}
		if !tc.allowZero && len(data) == 0 {
			t.Errorf("Marshal() returned empty bytes for %T", tc.msg)
		}
	}
}
