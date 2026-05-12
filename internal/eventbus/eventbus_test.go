package eventbus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Sample event types for testing
type UserCreated struct {
	UserID string
	Name   string
}

type OrderPlaced struct {
	OrderID string
	Amount  float64
}

// TestSubscribeAndPublish tests basic subscribe and async publish.
func TestSubscribeAndPublish(t *testing.T) {
	bus := New()

	var received UserCreated
	var mu sync.Mutex
	err := Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		mu.Lock()
		defer mu.Unlock()
		received = e
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(context.Background(), UserCreated{UserID: "123", Name: "Alice"})

	// Wait for async handler
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received.UserID != "123" {
		t.Errorf("expected UserID '123', got %q", received.UserID)
	}
	if received.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", received.Name)
	}
}

// TestPublishSync tests synchronous publish returns errors.
func TestPublishSync(t *testing.T) {
	bus := New()

	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		return errors.New("handler error")
	})

	err := bus.PublishSync(context.Background(), UserCreated{UserID: "1"})
	if err == nil {
		t.Error("expected error from PublishSync, got nil")
	}
	if err.Error() != "handler error" {
		t.Errorf("expected 'handler error', got %q", err.Error())
	}
}

// TestMultipleSubscribers tests that all subscribers receive the event.
func TestMultipleSubscribers(t *testing.T) {
	bus := New()

	var count int
	var mu sync.Mutex

	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})
	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})

	bus.PublishSync(context.Background(), UserCreated{})

	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2 handlers called, got %d", count)
	}
}

// TestNoSubscribers tests that publishing with no subscribers is a no-op.
func TestNoSubscribers(t *testing.T) {
	bus := New()

	// Should not panic or error
	bus.Publish(context.Background(), UserCreated{})
	err := bus.PublishSync(context.Background(), UserCreated{})
	if err != nil {
		t.Errorf("expected nil error for no subscribers, got %v", err)
	}
}

// TestMultipleEventTypes tests that different event types go to different handlers.
func TestMultipleEventTypes(t *testing.T) {
	bus := New()

	var userCalled, orderCalled bool
	var mu sync.Mutex

	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		mu.Lock()
		userCalled = true
		mu.Unlock()
		return nil
	})
	Subscribe(bus, func(ctx context.Context, e OrderPlaced) error {
		mu.Lock()
		orderCalled = true
		mu.Unlock()
		return nil
	})

	// Publish UserCreated
	bus.PublishSync(context.Background(), UserCreated{})

	mu.Lock()
	if !userCalled {
		t.Error("UserCreated handler should have been called")
	}
	if orderCalled {
		t.Error("OrderPlaced handler should NOT have been called")
	}
	mu.Unlock()

	// Reset and publish OrderPlaced
	mu.Lock()
	userCalled = false
	orderCalled = false
	mu.Unlock()

	bus.PublishSync(context.Background(), OrderPlaced{})

	mu.Lock()
	if userCalled {
		t.Error("UserCreated handler should NOT have been called")
	}
	if !orderCalled {
		t.Error("OrderPlaced handler should have been called")
	}
	mu.Unlock()
}

// TestShutdown tests graceful shutdown waits for handlers.
func TestShutdown(t *testing.T) {
	bus := New()

	var completed bool
	startCh := make(chan struct{})
	doneCh := make(chan struct{})

	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		close(startCh)
		time.Sleep(100 * time.Millisecond) // Simulate work
		completed = true
		close(doneCh)
		return nil
	})

	bus.Publish(context.Background(), UserCreated{})

	// Wait for handler to start
	<-startCh

	// Shutdown should wait for handler
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bus.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if !completed {
		t.Error("handler should have completed before shutdown returned")
	}
}

// TestShutdownTimeout tests that shutdown respects context timeout.
func TestShutdownTimeout(t *testing.T) {
	bus := New()

	Subscribe(bus, func(ctx context.Context, e UserCreated) error {
		time.Sleep(5 * time.Second) // Long work
		return nil
	})

	bus.Publish(context.Background(), UserCreated{})

	// Short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := bus.Shutdown(ctx)
	if err == nil {
		t.Error("expected timeout error from Shutdown")
	}
}

// TestSubscribeNilHandler tests that nil handler is rejected.
func TestSubscribeNilHandler(t *testing.T) {
	bus := New()

	var nilHandler func(context.Context, UserCreated) error
	err := Subscribe(bus, nilHandler)
	if err == nil {
		t.Error("expected error for nil handler")
	}
}
