package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestTimeout(t *testing.T) {
	t.Run("normal completion within timeout", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		m := Timeout(5 * time.Second)
		wrapped := m(handler)

		ctx := transport.NewContext(context.Background(), newTestTransporterWithHeaders("/test"))
		resp, err := wrapped(ctx, "req")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if resp != "success" {
			t.Errorf("expected 'success', got %v", resp)
		}
	})

	t.Run("timeout exceeded", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}

		m := Timeout(10 * time.Millisecond)
		wrapped := m(handler)

		ctx := transport.NewContext(context.Background(), newTestTransporterWithHeaders("/test"))
		_, err := wrapped(ctx, "req")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.Error)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if appErr.Code != 504 {
			t.Errorf("expected code 504, got %d", appErr.Code)
		}
	})

	t.Run("parent context cancelled", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}

		m := Timeout(5 * time.Second)
		wrapped := m(handler)

		parentCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		ctx := transport.NewContext(parentCtx, newTestTransporterWithHeaders("/test"))
		_, err := wrapped(ctx, "req")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("path-specific timeout", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}

		m := Timeout(5*time.Second,
			WithPathTimeout("/slow", 10*time.Millisecond),
		)
		wrapped := m(handler)

		ctx := transport.NewContext(context.Background(), newTestTransporterWithHeaders("/slow"))
		_, err := wrapped(ctx, "req")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		appErr, ok := err.(*errors.Error)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T", err)
		}
		if appErr.Code != 504 {
			t.Errorf("expected code 504, got %d", appErr.Code)
		}
	})
}

// testTransporter is a minimal transport.Transporter implementation for testing.
type testTransporter struct {
	operation   string
	reqHeader   transport.Header
	replyHeader transport.Header
}

// newTestTransporterWithHeaders creates a minimal transporter for testing.
func newTestTransporterWithHeaders(operation string) *testTransporter {
	return &testTransporter{
		operation:   operation,
		reqHeader:   &testHeader{headers: make(map[string]string)},
		replyHeader: &testHeader{headers: make(map[string]string)},
	}
}

func (t *testTransporter) Kind() transport.Kind               { return transport.KindHTTP }
func (t *testTransporter) Endpoint() string                   { return "localhost:8080" }
func (t *testTransporter) Operation() string                  { return t.operation }
func (t *testTransporter) RequestHeader() transport.Header    { return t.reqHeader }
func (t *testTransporter) ReplyHeader() transport.Header      { return t.replyHeader }
func (t *testTransporter) PathParams() map[string]string      { return nil }
func (t *testTransporter) QueryParams() map[string][]string   { return nil }

type testHeader struct {
	headers map[string]string
}

func (h *testHeader) Get(key string) string                  { return h.headers[key] }
func (h *testHeader) Set(key, value string)                  { h.headers[key] = value }
func (h *testHeader) Keys() []string {
	keys := make([]string, 0, len(h.headers))
	for k := range h.headers {
		keys = append(keys, k)
	}
	return keys
}
