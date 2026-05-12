package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestRequestID(t *testing.T) {
	t.Run("generates new request ID", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			id := RequestIDFromContext(ctx)
			if id == "" {
				t.Error("expected non-empty request ID")
			}
			if !strings.Contains(id, "-") {
				t.Error("expected UUID format request ID")
			}
			return "ok", nil
		}

		m := RequestID()
		wrapped := m(handler)

		tr := newTestTransporterWithHeaders("/test")
		ctx := transport.NewContext(context.Background(), tr)
		_, err := wrapped(ctx, "req")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify response header was set
		requestID := tr.ReplyHeader().Get(DefaultRequestIDHeader)
		if requestID == "" {
			t.Error("expected request ID in response header")
		}
	})

	t.Run("preserves existing request ID from header", func(t *testing.T) {
		expectedID := "test-request-id-12345"

		handler := func(ctx context.Context, req any) (any, error) {
			id := RequestIDFromContext(ctx)
			if id != expectedID {
				t.Errorf("expected %q, got %q", expectedID, id)
			}
			return "ok", nil
		}

		m := RequestID()
		wrapped := m(handler)

		tr := newTestTransporterWithHeaders("/test")
		tr.RequestHeader().Set(DefaultRequestIDHeader, expectedID)
		ctx := transport.NewContext(context.Background(), tr)
		_, err := wrapped(ctx, "req")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify response header
		requestID := tr.ReplyHeader().Get(DefaultRequestIDHeader)
		if requestID != expectedID {
			t.Errorf("expected %q, got %q", expectedID, requestID)
		}
	})

	t.Run("with custom header name", func(t *testing.T) {
		customHeader := "X-Trace-Id"

		handler := func(ctx context.Context, req any) (any, error) {
			id := RequestIDFromContext(ctx)
			if id == "" {
				t.Error("expected non-empty request ID")
			}
			return "ok", nil
		}

		m := RequestID(WithRequestIDHeader(customHeader))
		wrapped := m(handler)

		tr := newTestTransporterWithHeaders("/test")
		tr.RequestHeader().Set(customHeader, "custom-trace-id")
		ctx := transport.NewContext(context.Background(), tr)
		_, err := wrapped(ctx, "req")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify custom response header
		traceID := tr.ReplyHeader().Get(customHeader)
		if traceID != "custom-trace-id" {
			t.Errorf("expected %q, got %q", "custom-trace-id", traceID)
		}
	})

	t.Run("custom generator function", func(t *testing.T) {
		customID := "custom-generated-id"

		handler := func(ctx context.Context, req any) (any, error) {
			id := RequestIDFromContext(ctx)
			if id != customID {
				t.Errorf("expected %q, got %q", customID, id)
			}
			return "ok", nil
		}

		m := RequestID(WithRequestIDGenerator(func() string {
			return customID
		}))
		wrapped := m(handler)

		ctx := transport.NewContext(context.Background(), newTestTransporterWithHeaders("/test"))
		_, err := wrapped(ctx, "req")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("request ID functions", func(t *testing.T) {
		// Test NewContextWithRequestID and RequestIDFromContext
		id := "test-id"
		ctx := NewContextWithRequestID(context.Background(), id)
		got := RequestIDFromContext(ctx)
		if got != id {
			t.Errorf("expected %q, got %q", id, got)
		}

		// Test with no request ID
		empty := RequestIDFromContext(context.Background())
		if empty != "" {
			t.Errorf("expected empty string, got %q", empty)
		}
	})
}
