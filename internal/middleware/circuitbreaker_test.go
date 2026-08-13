package middleware

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/circuitbreaker"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestCircuitBreakerMiddleware(t *testing.T) {
	b := circuitbreaker.New(
		circuitbreaker.WithFailureCount(2),
		circuitbreaker.WithTimeout(30*time.Second),
	)
	m := CircuitBreaker(b)

	var handled int
	handler := m(func(ctx context.Context, req any) (any, error) {
		handled++
		return nil, stderrors.New("boom") // handler always fails
	})

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/orders",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	// Two failures trip the breaker; requests still reach the handler.
	for i := 0; i < 2; i++ {
		if _, err := handler(ctx, nil); err == nil {
			t.Fatalf("request %d: expected handler error", i)
		}
	}
	if handled != 2 {
		t.Fatalf("expected 2 handled requests, got %d", handled)
	}

	// Breaker open: third request fails fast without reaching the handler.
	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("expected error when circuit is open")
	}
	if handled != 2 {
		t.Fatalf("expected handler NOT called when circuit open, got %d handled", handled)
	}

	fwErr, ok := stderrors.AsType[*errors.Error](err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if fwErr.Code != errors.CodeServiceUnavailable {
		t.Errorf("expected code %d, got %d", errors.CodeServiceUnavailable, fwErr.Code)
	}
	if fwErr.Reason != "CIRCUIT_OPEN" {
		t.Errorf("expected reason CIRCUIT_OPEN, got %s", fwErr.Reason)
	}
}

func TestCircuitBreakerMiddleware_Recovers(t *testing.T) {
	b := circuitbreaker.New(
		circuitbreaker.WithFailureCount(1),
		circuitbreaker.WithTimeout(time.Second),
	)
	m := CircuitBreaker(b)

	var fail bool
	handler := m(func(ctx context.Context, req any) (any, error) {
		if fail {
			return nil, stderrors.New("boom")
		}
		return "ok", nil
	})

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	// Trip the breaker.
	fail = true
	if _, err := handler(ctx, nil); err == nil {
		t.Fatal("expected failure to trip breaker")
	}
	if _, err := handler(ctx, nil); err == nil {
		t.Fatal("expected rejection when open")
	}

	// After cool-down, the probe succeeds and the breaker recovers.
	fail = false
	time.Sleep(1100 * time.Millisecond)
	resp, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("expected probe success after cool-down, got %v", err)
	}
	if resp != "ok" {
		t.Errorf("expected response ok, got %v", resp)
	}
}
