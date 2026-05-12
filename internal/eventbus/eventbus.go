// Package eventbus provides a type-safe, in-process event bus for the Firefly framework.
//
// Basic usage:
//
//	bus := eventbus.New()
//	eventbus.Subscribe(bus, func(ctx context.Context, e UserCreated) error {
//	    // handle event
//	    return nil
//	})
//	bus.Publish(ctx, UserCreated{UserID: "123"})
package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics for event bus
var (
	eventsPublished = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "firefly_eventbus_published_total",
		Help: "Total number of events published",
	}, []string{"event"})

	handlerCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "firefly_eventbus_handler_calls_total",
		Help: "Total number of event handler invocations",
	}, []string{"event"})

	handlerErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "firefly_eventbus_handler_errors_total",
		Help: "Total number of event handler errors",
	}, []string{"event"})
)

func init() {
	prometheus.MustRegister(eventsPublished, handlerCalls, handlerErrors)
}

// Bus is an in-memory event bus that supports typed publish/subscribe.
type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]handlerEntry
	wg       sync.WaitGroup
	logger   *slog.Logger
}

// handlerEntry wraps a handler function.
type handlerEntry struct {
	fn func(ctx context.Context, event any) error
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{
		handlers: make(map[reflect.Type][]handlerEntry),
		logger:   slog.Default(),
	}
}

// NewWithLogger creates a new event bus with a custom logger.
func NewWithLogger(logger *slog.Logger) *Bus {
	return &Bus{
		handlers: make(map[reflect.Type][]handlerEntry),
		logger:   logger,
	}
}

// Publish asynchronously publishes an event. Returns immediately;
// handlers are invoked in background goroutines.
func (b *Bus) Publish(ctx context.Context, event any) {
	eventType := eventName(event)
	eventsPublished.WithLabelValues(eventType).Inc()

	b.mu.RLock()
	entries := b.handlers[reflect.TypeOf(event)]
	b.mu.RUnlock()

	for _, entry := range entries {
		b.wg.Add(1)
		go func(h func(ctx context.Context, event any) error) {
			defer b.wg.Done()
			handlerCalls.WithLabelValues(eventType).Inc()
			if err := h(ctx, event); err != nil {
				handlerErrors.WithLabelValues(eventType).Inc()
				b.logger.Error("event handler error",
					"event", eventType,
					"error", err,
				)
			}
		}(entry.fn)
	}
}

// PublishSync synchronously publishes an event. Blocks until all handlers complete.
// Returns the first non-nil handler error.
func (b *Bus) PublishSync(ctx context.Context, event any) error {
	eventType := eventName(event)
	eventsPublished.WithLabelValues(eventType).Inc()

	b.mu.RLock()
	entries := b.handlers[reflect.TypeOf(event)]
	b.mu.RUnlock()

	var firstErr error
	for _, entry := range entries {
		handlerCalls.WithLabelValues(eventType).Inc()
		if err := entry.fn(ctx, event); err != nil {
			handlerErrors.WithLabelValues(eventType).Inc()
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// eventName returns the type name of an event for metric labels.
func eventName(event any) string {
	return fmt.Sprintf("%T", event)
}

// Subscribe registers a typed event handler. The handler must have signature:
// func(context.Context, T) error where T is the event type.
//
// This is a package-level generic function. Call it as:
//
//	eventbus.Subscribe(bus, func(ctx context.Context, e MyEvent) error { ... })
func Subscribe[T any](b *Bus, handler func(context.Context, T) error) error {
	if handler == nil {
		return fmt.Errorf("eventbus: handler must not be nil")
	}

	var zero T
	eventType := reflect.TypeOf(zero)

	// Wrap the typed handler into the generic form
	wrapped := func(ctx context.Context, event any) error {
		typed, ok := event.(T)
		if !ok {
			return fmt.Errorf("eventbus: type mismatch for %T", event)
		}
		return handler(ctx, typed)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handlerEntry{fn: wrapped})

	b.logger.Debug("event handler registered", "event", eventType.String())
	return nil
}

// Start implements the app.Lifecycle interface. It is a no-op for the event bus
// because subscription registration happens at initialization time, not at startup.
func (b *Bus) Start(_ context.Context) error {
	b.logger.Info("event bus started")
	return nil
}

// Stop implements the app.Lifecycle interface. It delegates to Shutdown with a
// short timeout to drain in-flight async handlers.
func (b *Bus) Stop(_ context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return b.Shutdown(ctx)
}

// Shutdown waits for all in-flight async handlers to complete.
func (b *Bus) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("event bus shutdown complete")
		return nil
	case <-ctx.Done():
		b.logger.Warn("event bus shutdown timed out")
		return ctx.Err()
	}
}
