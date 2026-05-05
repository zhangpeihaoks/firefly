// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"math/rand"
	"testing"
	"testing/quick"

	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// TestProperty6_RecoveryPanicCapture tests property 6: Recovery 中间件 panic 捕获
// For any panic value, Recovery middleware should catch the panic and return an error response.
// **Validates: Requirements 3.7**
func TestProperty6_RecoveryPanicCapture(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(42)),
	}

	// Property: For any panic type, Recovery middleware should catch the panic
	// and return an error response (not nil response, error is not nil)
	f := func(panicType int) bool {
		// Ensure panicType is valid (0-3 for different panic types)
		panicType = panicType % 4

		// Create a handler that panics with different values based on panicType
		var panicHandler func(ctx context.Context, req any) (any, error)
		switch panicType {
		case 0:
			// String panic
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic("string panic")
			}
		case 1:
			// Error panic (using errors.Error)
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic(errors.New(500, "PANIC_ERROR", "panic error"))
			}
		case 2:
			// Integer panic
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic(42)
			}
		case 3:
			// Custom struct panic
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic(struct{ Message string }{Message: "custom panic"})
			}
		}

		// Wrap with Recovery middleware
		recovery := Recovery()
		wrapped := recovery(panicHandler)

		ctx := context.Background()
		resp, err := wrapped(ctx, "test request")

		// Property assertions:
		// 1. Response should be nil when panic occurs
		if resp != nil {
			t.Logf("Expected nil response when panic occurs, got %v", resp)
			return false
		}

		// 2. Error should not be nil (panic should be converted to error)
		if err == nil {
			t.Logf("Expected error when panic occurs, got nil")
			return false
		}

		// 3. Error should be an internal error (*errors.Error)
		appErr, ok := err.(*errors.Error)
		if !ok {
			t.Logf("Expected *errors.Error, got %T", err)
			return false
		}

		// 4. Error code should be 500 (Internal Server Error)
		if appErr.Code != 500 {
			t.Logf("Expected error code 500, got %d", appErr.Code)
			return false
		}

		// 5. Error reason should be INTERNAL_ERROR
		if appErr.Reason != "INTERNAL_ERROR" {
			t.Logf("Expected reason INTERNAL_ERROR, got %s", appErr.Reason)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 6 failed: %v", err)
	}
}

// TestProperty6_RecoveryPreservesNormalFlow tests that Recovery middleware
// does not interfere with normal (non-panic) handler execution.
// **Validates: Requirements 3.7**
func TestProperty6_RecoveryPreservesNormalFlow(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(42)),
	}

	// Property: For normal (non-panicking) handlers, Recovery should pass through
	// the response and error without modification.
	f := func(responseType int, shouldError bool) bool {
		responseType = responseType % 4

		var expectedResp any
		var expectedErr error

		// Create handler based on responseType
		var handler func(ctx context.Context, req any) (any, error)
		switch responseType {
		case 0:
			// String response
			expectedResp = "success response"
			expectedErr = nil
			handler = func(ctx context.Context, req any) (any, error) {
				return "success response", nil
			}
		case 1:
			// Integer response
			expectedResp = 123
			expectedErr = nil
			handler = func(ctx context.Context, req any) (any, error) {
				return 123, nil
			}
		case 2:
			// Map response
			expectedResp = map[string]string{"key": "value"}
			expectedErr = nil
			handler = func(ctx context.Context, req any) (any, error) {
				return map[string]string{"key": "value"}, nil
			}
		case 3:
			// Error response (without panic) - only when shouldError is true
			expectedResp = nil
			if shouldError {
				expectedErr = errors.New(400, "BAD_REQUEST", "bad request")
			}
			handler = func(ctx context.Context, req any) (any, error) {
				if shouldError {
					return nil, errors.New(400, "BAD_REQUEST", "bad request")
				}
				return "ok", nil
			}
		}

		// Wrap with Recovery middleware
		recovery := Recovery()
		wrapped := recovery(handler)

		ctx := context.Background()
		resp, err := wrapped(ctx, "test request")

		// Property assertions:
		// 1. Response should match expected
		if resp != expectedResp {
			// For map comparison, we need to check if both are nil or equal
			if !(resp == nil && expectedResp == nil) {
				t.Logf("Expected response %v, got %v", expectedResp, resp)
				return false
			}
		}

		// 2. Error should match expected
		if err != expectedErr {
			// If both are errors, compare codes
			if err != nil && expectedErr != nil {
				appErr, ok := err.(*errors.Error)
				expectedAppErr, ok2 := expectedErr.(*errors.Error)
				if !ok || !ok2 || appErr.Code != expectedAppErr.Code {
					t.Logf("Expected error %v, got %v", expectedErr, err)
					return false
				}
			} else if err == nil && expectedErr == nil {
				// Both nil, that's fine
			} else {
				t.Logf("Expected error %v, got %v", expectedErr, err)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 6 (normal flow) failed: %v", err)
	}
}

// TestProperty6_RecoveryMultipleCalls tests that Recovery middleware
// can handle multiple panic calls without losing effectiveness.
// **Validates: Requirements 3.7**
func TestProperty6_RecoveryMultipleCalls(t *testing.T) {
	config := &quick.Config{
		MaxCount: 20,
		Rand:     rand.New(rand.NewSource(42)),
	}

	// Property: Recovery middleware should continue to catch panics
	// across multiple invocations.
	f := func(callCount int) bool {
		if callCount <= 0 {
			callCount = 1
		}
		// Limit call count to prevent test from taking too long
		if callCount > 10 {
			callCount = 10
		}

		panicCount := 0

		handler := func(ctx context.Context, req any) (any, error) {
			panicCount++
			panic("multiple panic test")
		}

		recovery := Recovery()
		wrapped := recovery(handler)

		ctx := context.Background()

		// Call the wrapped handler multiple times
		for i := 0; i < callCount; i++ {
			resp, err := wrapped(ctx, "test request")

			// Each call should return error (not panic escaping)
			if err == nil {
				t.Logf("Call %d: expected error, got nil", i+1)
				return false
			}

			// Response should be nil
			if resp != nil {
				t.Logf("Call %d: expected nil response, got %v", i+1, resp)
				return false
			}
		}

		// Verify panic count matches call count
		if panicCount != callCount {
			t.Logf("Expected %d panics, got %d", callCount, panicCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 6 (multiple calls) failed: %v", err)
	}
}

// TestProperty6_RecoveryErrorTypes tests that Recovery middleware
// handles different error types correctly.
// **Validates: Requirements 3.7**
func TestProperty6_RecoveryErrorTypes(t *testing.T) {
	// Test with predefined panic types that are common in Go applications
	testCases := []struct {
		name     string
		panicVal any
	}{
		{
			name:     "nil panic",
			panicVal: nil,
		},
		{
			name:     "empty string panic",
			panicVal: "",
		},
		{
			name:     "string with message",
			panicVal: "something went wrong",
		},
		{
			name:     "negative integer panic",
			panicVal: -1,
		},
		{
			name:     "zero panic",
			panicVal: 0,
		},
		{
			name:     "positive integer panic",
			panicVal: 100,
		},
		{
			name:     "float panic",
			panicVal: 3.14,
		},
		{
			name:     "boolean true panic",
			panicVal: true,
		},
		{
			name:     "boolean false panic",
			panicVal: false,
		},
		{
			name:     "slice panic",
			panicVal: []int{1, 2, 3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(ctx context.Context, req any) (any, error) {
				panic(tc.panicVal)
			}

			recovery := Recovery()
			wrapped := recovery(handler)

			ctx := context.Background()
			resp, err := wrapped(ctx, "test request")

			// All panics should be caught and converted to error
			if resp != nil {
				t.Errorf("expected nil response, got %v", resp)
			}

			if err == nil {
				t.Error("expected error, got nil")
			}

			// Error should be internal error
			if err != nil {
				appErr, ok := err.(*errors.Error)
				if !ok {
					t.Errorf("expected *errors.Error, got %T", err)
				} else if appErr.Code != 500 {
					t.Errorf("expected code 500, got %d", appErr.Code)
				}
			}
		})
	}
}

// TestProperty6_RecoveryCustomHandler tests that custom panic handler is called
// when Recovery middleware catches a panic.
// **Validates: Requirements 3.7**
func TestProperty6_RecoveryCustomHandler(t *testing.T) {
	config := &quick.Config{
		MaxCount: 20,
		Rand:     rand.New(rand.NewSource(42)),
	}

	// Property: Custom handler should be called with the panic value
	f := func(panicType int) bool {
		panicType = panicType % 3

		var handlerCalled bool
		var capturedPanic any

		customHandler := func(ctx context.Context, req, err any) {
			handlerCalled = true
			capturedPanic = err
		}

		var panicHandler func(ctx context.Context, req any) (any, error)
		var expectedPanic any

		switch panicType {
		case 0:
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic("string panic")
			}
			expectedPanic = "string panic"
		case 1:
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic(42)
			}
			expectedPanic = 42
		case 2:
			panicHandler = func(ctx context.Context, req any) (any, error) {
				panic(struct{ Name string }{Name: "test"})
			}
			expectedPanic = struct{ Name string }{Name: "test"}
		}

		recovery := Recovery(WithRecoveryHandler(customHandler))
		wrapped := recovery(panicHandler)

		ctx := context.Background()
		resp, err := wrapped(ctx, "test request")

		// Custom handler should have been called
		if !handlerCalled {
			t.Log("expected custom handler to be called")
			return false
		}

		// Panic value should be captured
		if capturedPanic != expectedPanic {
			t.Logf("expected panic value %v, got %v", expectedPanic, capturedPanic)
			return false
		}

		// Should still return error
		if err == nil {
			t.Log("expected error, got nil")
			return false
		}

		// Response should be nil
		if resp != nil {
			t.Logf("expected nil response, got %v", resp)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 6 (custom handler) failed: %v", err)
	}
}
