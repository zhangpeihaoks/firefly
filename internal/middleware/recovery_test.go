// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// TestRecovery tests that Recovery middleware catches panics and returns an error.
func TestRecovery(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7
	// Property 6: Recovery 中间件 panic 捕获

	panicMsg := "test panic"
	handler := func(ctx context.Context, req any) (any, error) {
		panic(panicMsg)
	}

	// Wrap with Recovery middleware
	recovery := Recovery()
	wrapped := recovery(handler)

	ctx := context.Background()
	resp, err := wrapped(ctx, "request")

	// Should return nil response
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}

	// Should return an error
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Error should be an internal error
	if err != nil {
		appErr, ok := err.(*errors.Error)
		if !ok {
			t.Errorf("expected *errors.Error, got %T", err)
		} else {
			if appErr.Code != 500 {
				t.Errorf("expected code 500, got %d", appErr.Code)
			}
			if appErr.Reason != "INTERNAL_ERROR" {
				t.Errorf("expected reason INTERNAL_ERROR, got %s", appErr.Reason)
			}
		}
	}
}

// TestRecoveryNoPanic tests that Recovery middleware passes through normal requests.
func TestRecoveryNoPanic(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	expectedResp := "test response"
	handler := func(ctx context.Context, req any) (any, error) {
		return expectedResp, nil
	}

	// Wrap with Recovery middleware
	recovery := Recovery()
	wrapped := recovery(handler)

	ctx := context.Background()
	resp, err := wrapped(ctx, "request")

	// Should return the expected response
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if resp != expectedResp {
		t.Errorf("expected %v, got %v", expectedResp, resp)
	}
}

// TestRecoveryWithError tests that Recovery middleware passes through errors.
func TestRecoveryWithError(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	expectedErr := errors.New(400, "BAD_REQUEST", "bad request")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, expectedErr
	}

	// Wrap with Recovery middleware
	recovery := Recovery()
	wrapped := recovery(handler)

	ctx := context.Background()
	resp, err := wrapped(ctx, "request")

	// Should return nil response
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}

	// Should return the expected error
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestRecoveryWithCustomHandler tests that custom handler is called on panic.
func TestRecoveryWithCustomHandler(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	var handlerCalled bool
	var capturedPanic any
	var capturedReq any

	customHandler := func(ctx context.Context, req, err any) {
		handlerCalled = true
		capturedPanic = err
		capturedReq = req
	}

	panicValue := "custom panic"
	handler := func(ctx context.Context, req any) (any, error) {
		panic(panicValue)
	}

	// Wrap with Recovery middleware and custom handler
	recovery := Recovery(WithRecoveryHandler(customHandler))
	wrapped := recovery(handler)

	ctx := context.Background()
	req := "test request"
	_, err := wrapped(ctx, req)

	// Custom handler should have been called
	if !handlerCalled {
		t.Error("expected custom handler to be called")
	}

	// Panic value should be captured
	if capturedPanic != panicValue {
		t.Errorf("expected panic value %v, got %v", panicValue, capturedPanic)
	}

	// Request should be captured
	if capturedReq != req {
		t.Errorf("expected request %v, got %v", req, capturedReq)
	}

	// Should still return an error
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// TestRecoveryWithCustomLogger tests that custom logger is used.
func TestRecoveryWithCustomLogger(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	// Create a custom logger that writes to stderr
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	handler := func(ctx context.Context, req any) (any, error) {
		panic("logger test panic")
	}

	// Wrap with Recovery middleware and custom logger
	recovery := Recovery(WithRecoveryLogger(logger))
	wrapped := recovery(handler)

	ctx := context.Background()
	_, err := wrapped(ctx, "request")

	// Should return an error
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// TestRecoveryPanicWithDifferentTypes tests recovery from different panic types.
func TestRecoveryPanicWithDifferentTypes(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	tests := []struct {
		name      string
		panicVal  any
		wantError bool
	}{
		{
			name:      "string panic",
			panicVal:  "string panic",
			wantError: true,
		},
		{
			name:      "error panic",
			panicVal:  errors.New(400, "TEST_ERROR", "test error"),
			wantError: true,
		},
		{
			name:      "int panic",
			panicVal:  42,
			wantError: true,
		},
		{
			name:      "struct panic",
			panicVal:  struct{ Name string }{Name: "test"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(ctx context.Context, req any) (any, error) {
				panic(tt.panicVal)
			}

			recovery := Recovery()
			wrapped := recovery(handler)

			ctx := context.Background()
			_, err := wrapped(ctx, "request")

			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v", tt.wantError, err != nil)
			}
		})
	}
}

// TestRecoveryInChain tests that Recovery works correctly in a middleware chain.
func TestRecoveryInChain(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7
	// Note: When a panic occurs, the middleware chain is interrupted.
	// The "after" parts of middleware are not executed because the panic
	// breaks the normal control flow. Recovery catches the panic and returns
	// an error, but the deferred cleanup in outer middleware still runs.

	var order []string

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

	handler := func(ctx context.Context, req any) (any, error) {
		order = append(order, "handler")
		panic("chain panic")
	}

	// Chain middleware with Recovery at the beginning
	chain := Chain(Recovery(), m1, m2)
	wrapped := chain(handler)

	ctx := context.Background()
	_, err := wrapped(ctx, "request")

	// Should return an error
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify execution order - panic interrupts the chain
	// The panic happens in handler, so m2-after and m1-after are not called
	expected := []string{
		"m1-before", "m2-before", "handler",
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

// TestRecoveryMultiplePanics tests that Recovery can handle multiple panics.
func TestRecoveryMultiplePanics(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	panicCount := 0
	handler := func(ctx context.Context, req any) (any, error) {
		panicCount++
		panic("multiple panic test")
	}

	recovery := Recovery()
	wrapped := recovery(handler)

	ctx := context.Background()

	// Call multiple times
	for i := 0; i < 3; i++ {
		_, err := wrapped(ctx, "request")
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	// Should have panicked 3 times
	if panicCount != 3 {
		t.Errorf("expected 3 panics, got %d", panicCount)
	}
}

// TestRecoveryPreservesContext tests that context is preserved through recovery.
func TestRecoveryPreservesContext(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	type ctxKey struct{}
	expectedVal := "context-value"

	var capturedCtx context.Context

	customHandler := func(ctx context.Context, req, err any) {
		capturedCtx = ctx
	}

	handler := func(ctx context.Context, req any) (any, error) {
		panic("context test panic")
	}

	recovery := Recovery(WithRecoveryHandler(customHandler))
	wrapped := recovery(handler)

	ctx := context.WithValue(context.Background(), ctxKey{}, expectedVal)
	_, err := wrapped(ctx, "request")

	// Should return an error
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Context should be preserved
	if capturedCtx != nil {
		val := capturedCtx.Value(ctxKey{})
		if val != expectedVal {
			t.Errorf("expected context value %v, got %v", expectedVal, val)
		}
	}
}

// TestRecoveryStackInLog tests that stack trace is included in log output.
func TestRecoveryStackInLog(t *testing.T) {
	// Feature: backend-server-framework, Requirement 3.7

	// Create a pipe to capture log output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer r.Close()

	// Create logger that writes to the pipe
	logger := slog.New(slog.NewJSONHandler(w, nil))

	handler := func(ctx context.Context, req any) (any, error) {
		panic("stack trace test")
	}

	recovery := Recovery(WithRecoveryLogger(logger))
	wrapped := recovery(handler)

	ctx := context.Background()
	_, _ = wrapped(ctx, "request")

	// Close the writer to flush
	w.Close()

	// Read the log output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	logOutput := string(buf[:n])

	// Check that stack trace is included
	if !strings.Contains(logOutput, "stack") {
		t.Error("expected log to contain stack trace")
	}
}
