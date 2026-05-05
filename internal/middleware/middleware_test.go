// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"errors"
	"testing"
	"testing/quick"
)

// TestHandlerType tests that Handler type works correctly.
func TestHandlerType(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.1
	handler := func(ctx context.Context, req any) (any, error) {
		return "test response", nil
	}

	ctx := context.Background()
	resp, err := handler(ctx, "test request")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "test response" {
		t.Errorf("expected 'test response', got %v", resp)
	}
}

// TestMiddlewareType tests that Middleware type works correctly.
func TestMiddlewareType(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.2
	var called bool
	middleware := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			called = true
			return next(ctx, req)
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	wrapped := middleware(handler)
	ctx := context.Background()
	_, _ = wrapped(ctx, "request")

	if !called {
		t.Error("expected middleware to be called")
	}
}

// TestChain tests that Chain function composes middleware correctly.
func TestChain(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	var order []string

	// Create middleware that records execution order
	m1 := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			order = append(order, "m1-before")
			resp, err := next(ctx, req)
			order = append(order, "m1-after")
			return resp, err
		}
	}

	m2 := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			order = append(order, "m2-before")
			resp, err := next(ctx, req)
			order = append(order, "m2-after")
			return resp, err
		}
	}

	m3 := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			order = append(order, "m3-before")
			resp, err := next(ctx, req)
			order = append(order, "m3-after")
			return resp, err
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		order = append(order, "handler")
		return "response", nil
	}

	// Chain middleware
	chain := Chain(m1, m2, m3)
	wrapped := chain(handler)

	ctx := context.Background()
	_, _ = wrapped(ctx, "request")

	// Verify execution order: m1 -> m2 -> m3 -> handler -> m3 -> m2 -> m1
	expected := []string{
		"m1-before", "m2-before", "m3-before",
		"handler",
		"m3-after", "m2-after", "m1-after",
	}

	if len(order) != len(expected) {
		t.Errorf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}

	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected order[%d] = %s, got %v", i, v, order)
			break
		}
	}
}

// TestChainEmpty tests that Chain with no middleware returns the original handler.
func TestChainEmpty(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	chain := Chain()
	wrapped := chain(handler)

	ctx := context.Background()
	resp, err := wrapped(ctx, "request")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != "response" {
		t.Errorf("expected 'response', got %v", resp)
	}
}

// TestChainSingle tests that Chain with single middleware works correctly.
func TestChainSingle(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	var called bool

	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			called = true
			return next(ctx, req)
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	chain := Chain(m)
	wrapped := chain(handler)

	ctx := context.Background()
	_, _ = wrapped(ctx, "request")

	if !called {
		t.Error("expected middleware to be called")
	}
}

// TestChainErrorPropagation tests that errors are propagated through the chain.
func TestChainErrorPropagation(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	expectedErr := errors.New("test error")

	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, expectedErr
	}

	chain := Chain(m)
	wrapped := chain(handler)

	ctx := context.Background()
	_, err := wrapped(ctx, "request")

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestChainRequestModification tests that middleware can modify the request.
func TestChainRequestModification(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Modify request
			return next(ctx, "modified: "+req.(string))
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return req, nil
	}

	chain := Chain(m)
	wrapped := chain(handler)

	ctx := context.Background()
	resp, _ := wrapped(ctx, "request")

	if resp != "modified: request" {
		t.Errorf("expected 'modified: request', got %v", resp)
	}
}

// TestChainResponseModification tests that middleware can modify the response.
func TestChainResponseModification(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			resp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}
			return "modified: " + resp.(string), nil
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	chain := Chain(m)
	wrapped := chain(handler)

	ctx := context.Background()
	resp, _ := wrapped(ctx, "request")

	if resp != "modified: response" {
		t.Errorf("expected 'modified: response', got %v", resp)
	}
}

// TestChainContextPropagation tests that context is propagated through the chain.
func TestChainContextPropagation(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.3
	type ctxKey struct{}
	expectedVal := "context-value"

	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Add value to context
			ctx = context.WithValue(ctx, ctxKey{}, expectedVal)
			return next(ctx, req)
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		val := ctx.Value(ctxKey{})
		if val != expectedVal {
			t.Errorf("expected context value %v, got %v", expectedVal, val)
		}
		return val, nil
	}

	chain := Chain(m)
	wrapped := chain(handler)

	ctx := context.Background()
	resp, _ := wrapped(ctx, "request")

	if resp != expectedVal {
		t.Errorf("expected response %v, got %v", expectedVal, resp)
	}
}

// TestAdapt tests the Adapt function.
func TestAdapt(t *testing.T) {
	// Feature: backend-server-framework
	var called bool

	m := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			called = true
			return next(ctx, req)
		}
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Simple identity adapters
	toHandler := func(h Handler) any { return h }
	fromHandler := func(a any) Handler { return a.(Handler) }

	adapted := Adapt(m, toHandler, fromHandler)
	wrapped := adapted(handler)

	ctx := context.Background()
	_, _ = wrapped(ctx, "request")

	if !called {
		t.Error("expected middleware to be called via Adapt")
	}
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// Property 5: Middleware Chain Execution Order
// Validates: Requirements 3.3, 3.4
//
// For any middleware list, Chain function should execute middleware in registration order.
// The first middleware in the list should execute first (outermost), and the last middleware
// should execute last (innermost, just before the handler).
func TestProperty5MiddlewareChainOrder(t *testing.T) {
	// Test with different middleware counts (1-5)
	for count := 1; count <= 5; count++ {
		t.Run("", func(t *testing.T) {
			order := make([]int, 0, count)
			var ms []Middleware

			// Create middleware that records execution order
			for i := 0; i < count; i++ {
				i := i
				ms = append(ms, func(next Handler) Handler {
					return func(ctx context.Context, req any) (any, error) {
						order = append(order, i)
						return next(ctx, req)
					}
				})
			}

			// Create handler
			handler := func(ctx context.Context, req any) (any, error) {
				return nil, nil
			}

			// Chain middleware
			chain := Chain(ms...)
			wrapped := chain(handler)

			ctx := context.Background()
			_, _ = wrapped(ctx, nil)

			// Verify all middleware were called
			if len(order) != count {
				t.Fatalf("Expected %d middleware calls, got %d", count, len(order))
			}

			// Verify order is in registration order (0, 1, 2, ..., n-1)
			for i := 0; i < count; i++ {
				if i >= len(order) || order[i] != i {
					t.Errorf("Order mismatch at index %d: expected %d, got %v", i, i, order)
					return
				}
			}
		})
	}
}

// TestProperty5MiddlewareChainOrder_PBT is a property-based test that validates
// middleware execution order for arbitrary middleware counts using testing/quick.
//
// **Validates: Requirements 3.3, 3.4**
func TestProperty5MiddlewareChainOrder_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
	}

	f := func(count int) bool {
		// Limit count to reasonable range (1-10 middleware)
		if count < 1 || count > 10 {
			return true
		}

		order := make([]int, 0, count)
		var ms []Middleware

		// Create middleware that records execution order
		for i := 0; i < count; i++ {
			i := i
			ms = append(ms, func(next Handler) Handler {
				return func(ctx context.Context, req any) (any, error) {
					order = append(order, i)
					return next(ctx, req)
				}
			})
		}

		// Create handler
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, nil
		}

		// Chain middleware
		chain := Chain(ms...)
		wrapped := chain(handler)

		ctx := context.Background()
		_, _ = wrapped(ctx, nil)

		// Verify all middleware were called
		if len(order) != count {
			t.Logf("Expected %d middleware calls, got %d", count, len(order))
			return false
		}

		// Verify order is in registration order (0, 1, 2, ..., n-1)
		for i := 0; i < count; i++ {
			if order[i] != i {
				t.Logf("Order mismatch at index %d: expected %d, got %v", i, i, order)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 (middleware chain order PBT) failed: %v", err)
	}
}

// TestProperty5MiddlewareChainOrderReverse_PBT validates that middleware executes
// in reverse order during the response phase (after handler execution).
//
// **Validates: Requirements 3.3, 3.4**
func TestProperty5MiddlewareChainOrderReverse_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
	}

	f := func(count int) bool {
		// Limit count to reasonable range (1-10 middleware)
		if count < 1 || count > 10 {
			return true
		}

		// Record both "before" (request phase) and "after" (response phase) order
		beforeOrder := make([]int, 0, count)
		afterOrder := make([]int, 0, count)
		var ms []Middleware

		// Create middleware that records execution order in both phases
		for i := 0; i < count; i++ {
			i := i
			ms = append(ms, func(next Handler) Handler {
				return func(ctx context.Context, req any) (any, error) {
					beforeOrder = append(beforeOrder, i)
					resp, err := next(ctx, req)
					afterOrder = append(afterOrder, i)
					return resp, err
				}
			})
		}

		// Create handler
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, nil
		}

		// Chain middleware
		chain := Chain(ms...)
		wrapped := chain(handler)

		ctx := context.Background()
		_, _ = wrapped(ctx, nil)

		// Verify before order: 0, 1, 2, ..., n-1 (first middleware runs first)
		for i := 0; i < count; i++ {
			if beforeOrder[i] != i {
				t.Logf("Before order mismatch at index %d: expected %d, got %v", i, i, beforeOrder)
				return false
			}
		}

		// Verify after order: n-1, n-2, ..., 0 (last middleware completes first)
		for i := 0; i < count; i++ {
			expectedAfter := count - 1 - i
			if afterOrder[i] != expectedAfter {
				t.Logf("After order mismatch at index %d: expected %d, got %v", i, expectedAfter, afterOrder)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 (reverse order PBT) failed: %v", err)
	}
}

// TestProperty5MiddlewareChainEmpty_PBT validates that Chain with no middleware
// returns the original handler unchanged.
//
// **Validates: Requirements 3.3**
func TestProperty5MiddlewareChainEmpty_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
	}

	f := func(seed int) bool {
		// Create handler that returns unique value based on seed
		handler := func(ctx context.Context, req any) (any, error) {
			return seed, nil
		}

		// Chain with empty middleware list
		chain := Chain()
		wrapped := chain(handler)

		ctx := context.Background()
		resp, err := wrapped(ctx, "request")

		// Should return same value as original handler
		if err != nil {
			t.Logf("unexpected error: %v", err)
			return false
		}
		if resp != seed {
			t.Logf("expected %d, got %v", seed, resp)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 (empty chain PBT) failed: %v", err)
	}
}

// TestProperty5MiddlewareChainErrorPropagation_PBT validates that errors
// propagate correctly through the middleware chain.
//
// **Validates: Requirements 3.3, 3.4**
func TestProperty5MiddlewareChainErrorPropagation_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
	}

	f := func(middlewareIndex int) bool {
		// middlewareIndex determines which middleware in the chain returns an error
		// 0 = first middleware, higher = later middleware
		if middlewareIndex < 0 || middlewareIndex > 5 {
			return true
		}

		count := middlewareIndex + 1
		expectedErr := "error from middleware"

		var ms []Middleware
		for i := 0; i < count; i++ {
			i := i
			ms = append(ms, func(next Handler) Handler {
				return func(ctx context.Context, req any) (any, error) {
					// This middleware returns error
					if i == middlewareIndex {
						return nil, errors.New(expectedErr)
					}
					return next(ctx, req)
				}
			})
		}

		// Handler that should never be reached
		handler := func(ctx context.Context, req any) (any, error) {
			t.Logf("handler should not be reached")
			return nil, errors.New("handler should not be reached")
		}

		chain := Chain(ms...)
		wrapped := chain(handler)

		ctx := context.Background()
		_, err := wrapped(ctx, "request")

		// Verify error is propagated
		if err == nil {
			t.Logf("expected error but got nil")
			return false
		}
		if err.Error() != expectedErr {
			t.Logf("expected error %q, got %v", expectedErr, err)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 (error propagation PBT) failed: %v", err)
	}
}
