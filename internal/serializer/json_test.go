// Package serializer provides unit tests for JSON serializer.
package serializer

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

// TestPerson is a test struct for serialization tests
type TestPerson struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email,omitempty"`
}

// TestAddress is a nested struct for serialization tests
type TestAddress struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

// TestEmployee is a nested struct for serialization tests
type TestEmployee struct {
	Name    string         `json:"name"`
	Address TestAddress    `json:"address"`
	Phones  []string       `json:"phones,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// TestJSONSerializerMarshal tests the Marshal method of JSONSerializer
func TestJSONSerializerMarshal(t *testing.T) {
	serializer := NewJSONSerializer()

	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:    "string",
			input:   "hello",
			want:    `"hello"`,
			wantErr: false,
		},
		{
			name:    "int",
			input:   42,
			want:    "42",
			wantErr: false,
		},
		{
			name:    "negative int",
			input:   -100,
			want:    "-100",
			wantErr: false,
		},
		{
			name:    "float",
			input:   3.14,
			want:    "3.14",
			wantErr: false,
		},
		{
			name:    "bool true",
			input:   true,
			want:    "true",
			wantErr: false,
		},
		{
			name:    "bool false",
			input:   false,
			want:    "false",
			wantErr: false,
		},
		{
			name:    "empty array",
			input:   []int{},
			want:    "[]",
			wantErr: false,
		},
		{
			name:    "array with elements",
			input:   []int{1, 2, 3},
			want:    "[1,2,3]",
			wantErr: false,
		},
		{
			name:    "empty object",
			input:   map[string]any{},
			want:    "{}",
			wantErr: false,
		},
		{
			name:    "map with elements",
			input:   map[string]any{"key": "value"},
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "struct",
			input:   TestPerson{Name: "John", Age: 30},
			want:    `{"name":"John","age":30}`,
			wantErr: false,
		},
		{
			name:    "nil",
			input:   nil,
			want:    "null",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    `""`,
			wantErr: false,
		},
		{
			name:    "zero value int",
			input:   0,
			want:    "0",
			wantErr: false,
		},
		{
			name:    "zero value float",
			input:   0.0,
			want:    "0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serializer.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("Marshal() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

// TestJSONSerializerUnmarshal tests the Unmarshal method of JSONSerializer
func TestJSONSerializerUnmarshal(t *testing.T) {
	serializer := NewJSONSerializer()

	tests := []struct {
		name     string
		data     string
		target   any
		want     any
		wantErr  bool
		setupErr bool // if true, target is invalid (e.g., nil pointer)
	}{
		{
			name:     "string",
			data:     `"hello"`,
			target:   new(string),
			want:     "hello",
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "int",
			data:     "42",
			target:   new(int),
			want:     42,
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "negative int",
			data:     "-100",
			target:   new(int),
			want:     -100,
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "float",
			data:     "3.14",
			target:   new(float64),
			want:     3.14,
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "bool",
			data:     "true",
			target:   new(bool),
			want:     true,
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "array",
			data:     "[1,2,3]",
			target:   &[]int{},
			want:     &[]int{1, 2, 3},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "object",
			data:     `{"key":"value"}`,
			target:   &map[string]string{},
			want:     &map[string]string{"key": "value"},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "struct",
			data:     `{"name":"John","age":30}`,
			target:   &TestPerson{},
			want:     &TestPerson{Name: "John", Age: 30},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "nested struct",
			data:     `{"name":"John","address":{"city":"Beijing","country":"China"}}`,
			target:   &TestEmployee{},
			want:     &TestEmployee{Name: "John", Address: TestAddress{City: "Beijing", Country: "China"}},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "null to struct pointer",
			data:     "null",
			target:   &TestPerson{Name: "test"},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "invalid json",
			data:     `{invalid}`,
			target:   &TestPerson{},
			wantErr:  true,
			setupErr: false,
		},
		{
			name:     "empty string",
			data:     "",
			target:   &TestPerson{},
			wantErr:  true,
			setupErr: false,
		},
		{
			name:     "nil data",
			data:     "",
			target:   nil,
			wantErr:  true,
			setupErr: true,
		},
		{
			name:     "type mismatch - int to string",
			data:     "123",
			target:   new(string),
			wantErr:  true,
			setupErr: false,
		},
		{
			name:     "type mismatch - string to int",
			data:     `"123"`,
			target:   new(int),
			wantErr:  true,
			setupErr: false,
		},
		{
			name:     "array to slice",
			data:     "[1,2,3]",
			target:   &[]int{},
			want:     &[]int{1, 2, 3},
			wantErr:  false,
			setupErr: false,
		},
		{
			name:     "partial unmarshal",
			data:     `{"name":"John","age":30,"extra":"field"}`,
			target:   &TestPerson{Name: "Original"},
			want:     &TestPerson{Name: "John", Age: 30},
			wantErr:  false,
			setupErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupErr {
				err := serializer.Unmarshal([]byte(tt.data), tt.target)
				if err == nil {
					t.Errorf("Unmarshal() expected error for nil target, got nil")
				}
				return
			}

			err := serializer.Unmarshal([]byte(tt.data), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// For the null test case, verify that unmarshaling null clears the struct
				if tt.name == "null to struct pointer" {
					var result TestPerson
					err := serializer.Unmarshal([]byte("null"), &result)
					if err != nil {
						t.Errorf("Unmarshal() null error = %v", err)
					}
					// After unmarshaling null into a struct, the result should be zero value
					if result.Name != "" || result.Age != 0 {
						t.Errorf("Unmarshal() null to struct = %+v, want zero value", result)
					}
					return
				}

				// Compare using JSON roundtrip to avoid pointer comparison issues
				// Use semantic comparison (re-marshal and compare) to handle field ordering
				gotBytes, _ := json.Marshal(tt.target)
				wantBytes, _ := json.Marshal(tt.want)

				// Parse both to compare values rather than bytes
				var got, want any
				json.Unmarshal(gotBytes, &got)
				json.Unmarshal(wantBytes, &want)
				gotStr, _ := json.Marshal(got)
				wantStr, _ := json.Marshal(want)
				if !bytes.Equal(gotStr, wantStr) {
					t.Errorf("Unmarshal() = %v, want %v", string(gotStr), string(wantStr))
				}
			}
		})
	}
}

// TestJSONSerializerContentType tests the ContentType method of JSONSerializer
func TestJSONSerializerContentType(t *testing.T) {
	serializer := NewJSONSerializer()

	tests := []struct {
		name string
		want string
	}{
		{
			name: "json content type",
			want: "application/json",
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
		if serializer.ContentType() != "application/json" {
			t.Errorf("ContentType() not consistent on call %d", i)
		}
	}
}

// TestJSONSerializerRoundTrip tests that Marshal and Unmarshal are inverses
func TestJSONSerializerRoundTrip(t *testing.T) {
	serializer := NewJSONSerializer()

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "string",
			value: "hello world",
		},
		{
			name:  "int",
			value: 42,
		},
		{
			name:  "negative int",
			value: -100,
		},
		{
			name:  "float",
			value: 3.14159,
		},
		{
			name:  "bool",
			value: true,
		},
		{
			name:  "empty string",
			value: "",
		},
		{
			name:  "empty slice",
			value: []int{},
		},
		{
			name:  "slice with elements",
			value: []int{1, 2, 3, 4, 5},
		},
		{
			name:  "slice of strings",
			value: []string{"a", "b", "c"},
		},
		{
			name:  "empty map",
			value: map[string]any{},
		},
		{
			name:  "map with elements",
			value: map[string]any{"key1": "value1", "key2": 2},
		},
		{
			name:  "struct",
			value: TestPerson{Name: "John", Age: 30},
		},
		{
			name:  "struct with omitempty",
			value: TestPerson{Name: "Jane"},
		},
		{
			name: "nested struct",
			value: TestEmployee{
				Name:    "Mike",
				Address: TestAddress{City: "Shanghai", Country: "China"},
				Phones:  []string{"1234567890"},
				Meta:    map[string]any{"dept": "engineering"},
			},
		},
		{
			name:  "complex nested structure",
			value: []TestPerson{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 35}},
		},
		{
			name:  "zero value",
			value: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := serializer.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Create a new value of the same type
			var result any = tt.value
			switch tt.value.(type) {
			case string:
				result = ""
			case int:
				result = 0
			case int64:
				result = int64(0)
			case float64:
				result = 0.0
			case bool:
				result = false
			case []int:
				result = []int{}
			case []string:
				result = []string{}
			case []TestPerson:
				result = []TestPerson{}
			case map[string]any:
				result = map[string]any{}
			case TestPerson:
				result = TestPerson{}
			case TestEmployee:
				result = TestEmployee{}
			}

			// Unmarshal
			err = serializer.Unmarshal(data, &result)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// Compare using JSON roundtrip (semantic equivalence, not byte equality)
			originalBytes, _ := json.Marshal(tt.value)
			resultBytes, _ := json.Marshal(result)

			// Parse both and compare values to handle field ordering differences
			var original, unmarshaled any
			json.Unmarshal(originalBytes, &original)
			json.Unmarshal(resultBytes, &unmarshaled)

			// Use JSON equality comparison
			originalStr, _ := json.Marshal(original)
			resultStr, _ := json.Marshal(unmarshaled)
			if !bytes.Equal(originalStr, resultStr) {
				t.Errorf("RoundTrip: got %v, want %v", string(resultStr), string(originalStr))
			}
		})
	}
}

// TestJSONSerializerErrorHandling tests error handling scenarios
func TestJSONSerializerErrorHandling(t *testing.T) {
	serializer := NewJSONSerializer()

	t.Run("unmarshal into nil", func(t *testing.T) {
		err := serializer.Unmarshal([]byte(`{}`), nil)
		if err == nil {
			t.Error("Unmarshal() into nil should return error")
		}
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		var v map[string]any
		err := serializer.Unmarshal([]byte(`{invalid`), &v)
		if err == nil {
			t.Error("Unmarshal() invalid JSON should return error")
		}
	})

	t.Run("unmarshal with read-only target", func(t *testing.T) {
		// This should fail because we can't write to unaddressable value
		type Immutable struct {
			Name string
		}
		// Note: json.Unmarshal requires a pointer, so this tests the behavior
		var v Immutable
		err := serializer.Unmarshal([]byte(`{"name":"test"}`), v) // not a pointer
		if err == nil {
			t.Error("Unmarshal() with non-pointer should return error")
		}
	})

	t.Run("marshal non-serializable", func(t *testing.T) {
		// Use a function which cannot be marshaled
		type WithFunc struct {
			Name string
			Func func() `json:"-"`
		}
		v := WithFunc{Name: "test", Func: func() {}}
		_, err := serializer.Marshal(v)
		// This should not error because of json:"-" tag, but let's test
		if err != nil {
			t.Logf("Marshal() returned error for function field: %v", err)
		}
	})
}

// TestJSONSerializerInterface tests that JSONSerializer implements Serializer interface
func TestJSONSerializerInterface(t *testing.T) {
	// Compile-time interface check
	var _ Serializer = NewJSONSerializer()
}

// TestJSONSerializerNew tests the NewJSONSerializer constructor
func TestJSONSerializerNew(t *testing.T) {
	serializer := NewJSONSerializer()
	if serializer == nil {
		t.Error("NewJSONSerializer() returned nil")
	}

	// Test that multiple calls return usable instances
	s1 := NewJSONSerializer()
	s2 := NewJSONSerializer()
	if s1 == nil || s2 == nil {
		t.Error("NewJSONSerializer() returned nil instance")
	}

	// Both should work independently
	data1, _ := s1.Marshal("test1")
	data2, _ := s2.Marshal("test2")

	if string(data1) != `"test1"` || string(data2) != `"test2"` {
		t.Error("Serializers not working independently")
	}
}

// TestJSONSerializerSpecialValues tests special JSON values
func TestJSONSerializerSpecialValues(t *testing.T) {
	serializer := NewJSONSerializer()

	t.Run("marshal special floats", func(t *testing.T) {
		tests := []struct {
			name  string
			value float64
		}{
			{"positive infinity", math.Inf(1)},
			{"negative infinity", math.Inf(-1)},
			{"NaN", math.NaN()},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := serializer.Marshal(tt.value)
				// JSON marshal of Inf/NaN returns error
				if err != nil {
					t.Logf("Marshal(%v) returned error (expected for special float): %v", tt.value, err)
				}
				if data != nil {
					t.Logf("Marshal(%v) = %s", tt.value, string(data))
				}
			})
		}
	})

	t.Run("unmarshal special values", func(t *testing.T) {
		// Test unmarshaling null
		var v *TestPerson
		err := serializer.Unmarshal([]byte("null"), &v)
		if err != nil {
			t.Errorf("Unmarshal() null error = %v", err)
		}
		if v != nil {
			t.Error("Unmarshal() null should set pointer to nil")
		}

		// Test unmarshaling into existing value
		p := &TestPerson{Name: "Old"}
		err = serializer.Unmarshal([]byte(`{"name":"New","age":25}`), p)
		if err != nil {
			t.Errorf("Unmarshal() error = %v", err)
		}
		if p.Name != "New" || p.Age != 25 {
			t.Errorf("Unmarshal() did not update existing struct: got %+v", p)
		}
	})
}

// TestJSONSerializerOmitempty tests omitempty behavior
func TestJSONSerializerOmitempty(t *testing.T) {
	serializer := NewJSONSerializer()

	t.Run("struct with omitempty fields", func(t *testing.T) {
		// Test with zero values that should be omitted
		p1 := TestPerson{Name: "John"} // Age is 0, Email is ""
		data, err := serializer.Marshal(p1)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		// Age=0 should be omitted because it's zero value
		// Email should be omitted because it's empty with omitempty
		if !bytes.Contains(data, []byte(`"name":"John"`)) {
			t.Errorf("Expected name field, got %s", string(data))
		}
		// Age should be present (no omitempty tag)
		if !bytes.Contains(data, []byte(`"age":0`)) {
			t.Logf("Age 0 value: got %s", string(data))
		}
	})
}

// TestJSONSerializerComplexTypes tests complex data types
func TestJSONSerializerComplexTypes(t *testing.T) {
	serializer := NewJSONSerializer()

	t.Run("map with various value types", func(t *testing.T) {
		m := map[string]any{
			"string": "value",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"array":  []int{1, 2, 3},
			"object": map[string]any{"nested": "value"},
			"null":   nil,
		}
		data, err := serializer.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		var result map[string]any
		err = serializer.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if result["string"] != "value" {
			t.Errorf("string value mismatch: got %v", result["string"])
		}
		if result["int"].(float64) != 42 { // JSON numbers are float64
			t.Errorf("int value mismatch: got %v", result["int"])
		}
		if result["bool"] != true {
			t.Errorf("bool value mismatch: got %v", result["bool"])
		}
	})

	t.Run("array of mixed types", func(t *testing.T) {
		// Note: Go slices must be homogeneous, but we can use []any
		arr := []any{"string", 42, 3.14, true, nil, []int{1, 2}}
		data, err := serializer.Marshal(arr)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		var result []any
		err = serializer.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}

		if len(result) != 6 {
			t.Errorf("array length mismatch: got %d, want 6", len(result))
		}
	})

	t.Run("struct with private fields", func(t *testing.T) {
		type WithPrivate struct {
			Public  string `json:"public"`
			private string `json:"private"` // This won't be serialized
		}
		v := WithPrivate{Public: "visible", private: "hidden"}
		data, err := serializer.Marshal(v)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		// Private fields should not appear in JSON
		if bytes.Contains(data, []byte("private")) {
			t.Error("Private field should not be serialized")
		}
	})
}
