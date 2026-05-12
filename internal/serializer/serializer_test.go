// Package serializer provides property-based tests for the serialization layer.
package serializer

import (
	"testing"
	"testing/quick"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestProperty44JSONSerializationCorrectness tests Property 44:
// 对于任何数据结构，JSON 序列化器应正确序列化和反序列化
//
// Feature: backend-server-framework, Property 44: JSON 序列化正确性
// Validates: Design Document Property 44
func TestProperty44JSONSerializationCorrectness(t *testing.T) {
	// Feature: backend-server-framework, Property 44: JSON 序列化正确性
	serializer := NewJSONSerializer()

	// Test basic types
	t.Run("basic_types", func(t *testing.T) {
		// Test string round-trip
		stringProp := func(s string) bool {
			data, err := serializer.Marshal(s)
			if err != nil {
				return false
			}
			var result string
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result == s
		}
		if err := quick.Check(stringProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("string round-trip property failed: %v", err)
		}

		// Test int round-trip
		intProp := func(n int) bool {
			data, err := serializer.Marshal(n)
			if err != nil {
				return false
			}
			var result int
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result == n
		}
		if err := quick.Check(intProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("int round-trip property failed: %v", err)
		}

		// Test float64 round-trip
		floatProp := func(n float64) bool {
			data, err := serializer.Marshal(n)
			if err != nil {
				return false
			}
			var result float64
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result == n
		}
		if err := quick.Check(floatProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("float64 round-trip property failed: %v", err)
		}

		// Test bool round-trip
		boolProp := func(b bool) bool {
			data, err := serializer.Marshal(b)
			if err != nil {
				return false
			}
			var result bool
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result == b
		}
		if err := quick.Check(boolProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("bool round-trip property failed: %v", err)
		}
	})

	// Test composite types
	t.Run("composite_types", func(t *testing.T) {
		// Test slice of int round-trip
		sliceProp := func(s []int) bool {
			// Skip nil slices as JSON unmarshal produces empty slice
			if s == nil {
				return true
			}
			data, err := serializer.Marshal(s)
			if err != nil {
				return false
			}
			result := make([]int, 0)
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			if len(result) != len(s) {
				return false
			}
			for i := range s {
				if result[i] != s[i] {
					return false
				}
			}
			return true
		}
		if err := quick.Check(sliceProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("slice round-trip property failed: %v", err)
		}

		// Test map[string]int round-trip
		mapProp := func(m map[string]int) bool {
			// Skip nil maps as JSON unmarshal produces empty map
			if m == nil {
				return true
			}
			data, err := serializer.Marshal(m)
			if err != nil {
				return false
			}
			result := make(map[string]int)
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			if len(result) != len(m) {
				return false
			}
			for k, v := range m {
				if result[k] != v {
					return false
				}
			}
			return true
		}
		if err := quick.Check(mapProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("map round-trip property failed: %v", err)
		}
	})

	// Test struct types
	t.Run("struct_types", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		structProp := func(name string, value int) bool {
			original := TestStruct{Name: name, Value: value}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			var result TestStruct
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result.Name == original.Name && result.Value == original.Value
		}
		if err := quick.Check(structProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("struct round-trip property failed: %v", err)
		}
	})

	// Test nested struct types
	t.Run("nested_struct_types", func(t *testing.T) {
		type Inner struct {
			Value string `json:"value"`
		}
		type Outer struct {
			Name  string `json:"name"`
			Inner Inner  `json:"inner"`
		}

		nestedProp := func(name, innerValue string) bool {
			original := Outer{
				Name:  name,
				Inner: Inner{Value: innerValue},
			}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			var result Outer
			if err := serializer.Unmarshal(data, &result); err != nil {
				return false
			}
			return result.Name == original.Name && result.Inner.Value == original.Inner.Value
		}
		if err := quick.Check(nestedProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("nested struct round-trip property failed: %v", err)
		}
	})
}

// TestProperty46SerializationModeSwitching tests Property 46:
// 对于任何序列化模式配置，工厂应返回正确的序列化器实例
//
// Feature: backend-server-framework, Property 46: 序列化模式切换
// Validates: Design Document Property 46
func TestProperty46SerializationModeSwitching(t *testing.T) {
	// Feature: backend-server-framework, Property 46: 序列化模式切换

	// Test that NewFactory creates factory with correct default mode
	t.Run("factory_default_mode", func(t *testing.T) {
		modes := []SerializerMode{ModeJSON, ModeProtobuf}
		for _, mode := range modes {
			factory := NewFactory(mode)
			if factory.Mode() != mode {
				t.Errorf("expected mode %s, got %s", mode, factory.Mode())
			}
		}
	})

	// Test that GetSerializer returns correct serializer for each mode
	t.Run("get_serializer_by_mode", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		// Test JSON mode
		jsonSerializer, err := factory.GetSerializerByMode(ModeJSON)
		if err != nil {
			t.Errorf("unexpected error getting JSON serializer: %v", err)
		}
		if _, ok := jsonSerializer.(*JSONSerializer); !ok {
			t.Error("expected JSONSerializer for ModeJSON")
		}

		// Test Protobuf mode
		protobufSerializer, err := factory.GetSerializerByMode(ModeProtobuf)
		if err != nil {
			t.Errorf("unexpected error getting Protobuf serializer: %v", err)
		}
		if _, ok := protobufSerializer.(*ProtobufSerializer); !ok {
			t.Error("expected ProtobufSerializer for ModeProtobuf")
		}
	})

	// Test that SetMode correctly switches modes
	t.Run("set_mode_switches_serializer", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		// Verify initial mode is JSON
		if factory.Mode() != ModeJSON {
			t.Errorf("expected initial mode %s, got %s", ModeJSON, factory.Mode())
		}

		// Get initial serializer
		initialSerializer := factory.GetSerializer()
		if _, ok := initialSerializer.(*JSONSerializer); !ok {
			t.Error("expected initial serializer to be JSONSerializer")
		}

		// Switch to Protobuf mode
		if err := factory.SetMode(ModeProtobuf); err != nil {
			t.Errorf("unexpected error setting mode: %v", err)
		}

		// Verify mode changed
		if factory.Mode() != ModeProtobuf {
			t.Errorf("expected mode %s after SetMode, got %s", ModeProtobuf, factory.Mode())
		}

		// Get serializer after mode switch
		newSerializer := factory.GetSerializer()
		if _, ok := newSerializer.(*ProtobufSerializer); !ok {
			t.Error("expected serializer to be ProtobufSerializer after mode switch")
		}

		// Switch back to JSON mode
		if err := factory.SetMode(ModeJSON); err != nil {
			t.Errorf("unexpected error setting mode: %v", err)
		}

		// Verify mode changed back
		if factory.Mode() != ModeJSON {
			t.Errorf("expected mode %s after SetMode, got %s", ModeJSON, factory.Mode())
		}

		// Get serializer after mode switch back
		finalSerializer := factory.GetSerializer()
		if _, ok := finalSerializer.(*JSONSerializer); !ok {
			t.Error("expected serializer to be JSONSerializer after mode switch back")
		}
	})

	// Test that SetMode returns error for unknown mode
	t.Run("set_mode_unknown_mode_error", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		err := factory.SetMode(SerializerMode("unknown"))
		if err == nil {
			t.Error("expected error for unknown mode")
		}
	})

	// Test that GetSerializerByMode returns error for unknown mode
	t.Run("get_serializer_unknown_mode_error", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		_, err := factory.GetSerializerByMode(SerializerMode("unknown"))
		if err == nil {
			t.Error("expected error for unknown mode")
		}
	})

	// Test concurrent mode switching
	t.Run("concurrent_mode_switching", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		// Run multiple goroutines switching modes
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					mode := ModeJSON
					if id%2 == 0 {
						mode = ModeProtobuf
					}
					_ = factory.SetMode(mode)
					_ = factory.GetSerializer()
					_ = factory.Mode()
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestProperty47SerializerContentType tests Property 47:
// 对于任何序列化器，ContentType() 方法应返回正确的 Content-Type
//
// Feature: backend-server-framework, Property 47: 序列化 Content-Type
// Validates: Design Document Property 47
func TestProperty47SerializerContentType(t *testing.T) {
	// Feature: backend-server-framework, Property 47: 序列化 Content-Type

	// Test JSONSerializer ContentType
	t.Run("json_serializer_content_type", func(t *testing.T) {
		serializer := NewJSONSerializer()
		expectedContentType := "application/json"

		// Test that ContentType always returns the same value
		contentTypeProp := func(_ bool) bool {
			return serializer.ContentType() == expectedContentType
		}
		if err := quick.Check(contentTypeProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("JSON ContentType property failed: %v", err)
		}
	})

	// Test ProtobufSerializer ContentType
	t.Run("protobuf_serializer_content_type", func(t *testing.T) {
		serializer := NewProtobufSerializer()
		expectedContentType := "application/protobuf"

		// Test that ContentType always returns the same value
		contentTypeProp := func(_ bool) bool {
			return serializer.ContentType() == expectedContentType
		}
		if err := quick.Check(contentTypeProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("Protobuf ContentType property failed: %v", err)
		}
	})

	// Test ContentType through Factory
	t.Run("factory_content_type", func(t *testing.T) {
		factory := NewFactory(ModeJSON)

		// Test JSON mode ContentType
		jsonSerializer, err := factory.GetSerializerByMode(ModeJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if jsonSerializer.ContentType() != "application/json" {
			t.Errorf("expected application/json, got %s", jsonSerializer.ContentType())
		}

		// Test Protobuf mode ContentType
		protobufSerializer, err := factory.GetSerializerByMode(ModeProtobuf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if protobufSerializer.ContentType() != "application/protobuf" {
			t.Errorf("expected application/protobuf, got %s", protobufSerializer.ContentType())
		}
	})

	// Test that ContentType is idempotent
	t.Run("content_type_idempotent", func(t *testing.T) {
		jsonSerializer := NewJSONSerializer()
		protobufSerializer := NewProtobufSerializer()

		// Call ContentType multiple times and verify consistency
		for i := 0; i < 100; i++ {
			if jsonSerializer.ContentType() != "application/json" {
				t.Error("JSON ContentType not idempotent")
			}
			if protobufSerializer.ContentType() != "application/protobuf" {
				t.Error("Protobuf ContentType not idempotent")
			}
		}
	})
}

// TestProtobufSerializationCorrectness tests Property 45:
// 对于任何 proto.Message，Protobuf 序列化器应正确序列化和反序列化
//
// Feature: backend-server-framework, Property 45: Protobuf 序列化正确性
// Validates: Design Document Property 45
func TestProtobufSerializationCorrectness(t *testing.T) {
	// Feature: backend-server-framework, Property 45: Protobuf 序列化正确性
	serializer := NewProtobufSerializer()

	// Test error handling for non-proto.Message types
	t.Run("non_proto_message_error", func(t *testing.T) {
		nonProtoValues := []any{
			"string",
			42,
			3.14,
			true,
			[]int{1, 2, 3},
			map[string]string{"key": "value"},
			struct{ Name string }{Name: "test"},
		}

		for _, v := range nonProtoValues {
			_, err := serializer.Marshal(v)
			if err == nil {
				t.Errorf("expected error for non-proto.Message type %T", v)
			}

			err = serializer.Unmarshal([]byte{}, v)
			if err == nil {
				t.Errorf("expected error for non-proto.Message type %T", v)
			}
		}
	})

	// Property-based test: round-trip serialization for any valid message
	t.Run("round_trip_property", func(t *testing.T) {
		// Test with wrapperspb.StringValue
		stringProp := func(s string) bool {
			original := &wrapperspb.StringValue{Value: s}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			result := &wrapperspb.StringValue{}
			if err := serializer.Unmarshal(data, result); err != nil {
				return false
			}
			return result.Value == s
		}
		if err := quick.Check(stringProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("StringValue round-trip property failed: %v", err)
		}

		// Test with wrapperspb.Int32Value
		intProp := func(n int32) bool {
			original := &wrapperspb.Int32Value{Value: n}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			result := &wrapperspb.Int32Value{}
			if err := serializer.Unmarshal(data, result); err != nil {
				return false
			}
			return result.Value == n
		}
		if err := quick.Check(intProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("Int32Value round-trip property failed: %v", err)
		}

		// Test with wrapperspb.BoolValue
		boolProp := func(b bool) bool {
			original := &wrapperspb.BoolValue{Value: b}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			result := &wrapperspb.BoolValue{}
			if err := serializer.Unmarshal(data, result); err != nil {
				return false
			}
			return result.Value == b
		}
		if err := quick.Check(boolProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("BoolValue round-trip property failed: %v", err)
		}

		// Test with wrapperspb.BytesValue
		bytesProp := func(b []byte) bool {
			original := &wrapperspb.BytesValue{Value: b}
			data, err := serializer.Marshal(original)
			if err != nil {
				return false
			}
			result := &wrapperspb.BytesValue{}
			if err := serializer.Unmarshal(data, result); err != nil {
				return false
			}
			return string(result.Value) == string(b)
		}
		if err := quick.Check(bytesProp, &quick.Config{MaxCount: 100}); err != nil {
			t.Errorf("BytesValue round-trip property failed: %v", err)
		}
	})
}

// TestSerializerInterface tests that all serializers implement the Serializer interface
func TestSerializerInterface(t *testing.T) {
	// Compile-time interface check
	var _ Serializer = NewJSONSerializer()
	var _ Serializer = NewProtobufSerializer()
}

// TestFactoryRegister tests custom serializer registration
func TestFactoryRegister(t *testing.T) {
	factory := NewFactory(ModeJSON)

	// Create a custom serializer
	customSerializer := &mockSerializer{contentType: "application/custom"}

	// Register custom serializer
	factory.Register("custom", customSerializer)

	// Verify it can be retrieved
	retrieved, err := factory.GetSerializerByMode("custom")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if retrieved != customSerializer {
		t.Error("retrieved serializer does not match registered")
	}

	// Verify we can switch to custom mode
	if err := factory.SetMode("custom"); err != nil {
		t.Errorf("unexpected error setting custom mode: %v", err)
	}
	if factory.Mode() != "custom" {
		t.Errorf("expected mode custom, got %s", factory.Mode())
	}
}

// mockSerializer is a mock implementation for testing
type mockSerializer struct {
	contentType string
}

func (m *mockSerializer) Marshal(v any) ([]byte, error) {
	return []byte("mock"), nil
}

func (m *mockSerializer) Unmarshal(data []byte, v any) error {
	return nil
}

func (m *mockSerializer) ContentType() string {
	return m.contentType
}

// TestProperty44JSONNilHandling tests edge cases for nil values
func TestProperty44JSONNilHandling(t *testing.T) {
	// Feature: backend-server-framework, Property 44: JSON 序列化正确性
	serializer := NewJSONSerializer()

	t.Run("nil_input", func(t *testing.T) {
		data, err := serializer.Marshal(nil)
		if err != nil {
			t.Errorf("unexpected error marshaling nil: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("expected 'null', got %s", string(data))
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		var emptySlice []int
		data, err := serializer.Marshal(emptySlice)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("expected 'null' for nil slice, got %s", string(data))
		}
	})

	t.Run("empty_map", func(t *testing.T) {
		var emptyMap map[string]int
		data, err := serializer.Marshal(emptyMap)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("expected 'null' for nil map, got %s", string(data))
		}
	})
}

// TestProperty46FactoryModeConsistency tests that factory mode operations are consistent
func TestProperty46FactoryModeConsistency(t *testing.T) {
	// Feature: backend-server-framework, Property 46: 序列化模式切换
	factory := NewFactory(ModeJSON)

	// Property: Mode() should always return the last successfully set mode
	modeProp := func(mode SerializerMode) bool {
		// Only test valid modes
		if mode != ModeJSON && mode != ModeProtobuf {
			return true // Skip invalid modes
		}

		if err := factory.SetMode(mode); err != nil {
			return false
		}
		return factory.Mode() == mode
	}

	if err := quick.Check(modeProp, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("mode consistency property failed: %v", err)
	}
}

// TestProperty47ContentTypeConsistency tests ContentType consistency across serializers
func TestProperty47ContentTypeConsistency(t *testing.T) {
	// Feature: backend-server-framework, Property 47: 序列化 Content-Type

	// Property: All JSONSerializers should return "application/json"
	jsonProp := func(_ int) bool {
		s := NewJSONSerializer()
		return s.ContentType() == "application/json"
	}
	if err := quick.Check(jsonProp, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("JSON ContentType consistency property failed: %v", err)
	}

	// Property: All ProtobufSerializers should return "application/protobuf"
	protobufProp := func(_ int) bool {
		s := NewProtobufSerializer()
		return s.ContentType() == "application/protobuf"
	}
	if err := quick.Check(protobufProp, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Protobuf ContentType consistency property failed: %v", err)
	}
}

// init is used to ensure proto package is imported for interface checks
var _ proto.Message = nil
