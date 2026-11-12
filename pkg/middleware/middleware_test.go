package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	ferrors "github.com/zhangpeihaoks/firefly/pkg/errors"
)

// TestChainOrder verifies the documented execution order: the first
// middleware in the chain wraps the rest, so it runs first on the way in and
// last on the way out.
func TestChainOrder(t *testing.T) {
	var order []string
	// recorder returns a middleware that appends name on entry and name+":out"
	// on exit, so both directions of the onion are observable.
	recorder := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(ctx context.Context, req any) (any, error) {
				order = append(order, name)
				resp, err := next(ctx, req)
				order = append(order, name+":out")
				return resp, err
			}
		}
	}

	h := Chain(recorder("first"), recorder("second"))(func(_ context.Context, _ any) (any, error) {
		order = append(order, "handler")
		return "ok", nil
	})

	if _, err := h(context.Background(), nil); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	want := []string{"first", "second", "handler", "second:out", "first:out"}
	if len(order) != len(want) {
		t.Fatalf("order length = %d, want %d (%v)", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], want[i], order)
		}
	}
}

// TestRecoveryTurnsPanicIntoError verifies a panicking handler surfaces as an
// error to the caller instead of taking the process down.
func TestRecoveryTurnsPanicIntoError(t *testing.T) {
	h := Recovery()(func(_ context.Context, _ any) (any, error) {
		panic("boom")
	})

	resp, err := h(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected an error from a panicking handler, got response %v", resp)
	}
}

// TestRequestIDRoundTrip verifies the middleware stores a generated id in the
// context and that RequestIDFromContext can read it back.
func TestRequestIDRoundTrip(t *testing.T) {
	var seen string
	h := RequestID()(func(ctx context.Context, _ any) (any, error) {
		seen = RequestIDFromContext(ctx)
		return nil, nil
	})

	if _, err := h(context.Background(), nil); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if seen == "" {
		t.Fatal("RequestIDFromContext returned an empty id")
	}
}

// TestTimeoutReportsGatewayTimeout verifies that when the handler outlives the
// timeout, callers get the framework's gateway-timeout error (HTTP 504).
func TestTimeoutReportsGatewayTimeout(t *testing.T) {
	h := Timeout(20 * time.Millisecond)(func(_ context.Context, _ any) (any, error) {
		// Deliberately ignore the context and outlive the timeout by a wide
		// margin so the middleware always takes its timeout branch.
		time.Sleep(300 * time.Millisecond)
		return nil, nil
	})

	_, err := h(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error after the timeout")
	}
	var fe *ferrors.Error
	if !errors.As(err, &fe) {
		t.Fatalf("error = %v, want *errors.Error", err)
	}
	if fe.Code != ferrors.CodeGatewayTimeout {
		t.Fatalf("code = %d, want %d", fe.Code, ferrors.CodeGatewayTimeout)
	}
}

// TestTimeoutCancelsContext verifies the context handed to the handler is
// cancelled once the configured timeout elapses.
func TestTimeoutCancelsContext(t *testing.T) {
	observed := make(chan struct{})
	h := Timeout(20 * time.Millisecond)(func(ctx context.Context, _ any) (any, error) {
		select {
		case <-ctx.Done():
			close(observed)
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			return nil, nil
		}
	})

	// Either error is acceptable: the middleware and the handler observe the
	// same deadline, so which side reports first is a scheduling race.
	if _, err := h(context.Background(), nil); err == nil {
		t.Fatal("expected an error after the timeout")
	}
	select {
	case <-observed:
	default:
		t.Fatal("handler did not observe the cancelled context")
	}
}
