// Package alert provides webhook-based alerting for the Firefly framework.
//
// An AlertManager fans out alerts to one or more Notifiers (webhooks). Built-in
// formatters produce platform-native payloads for Feishu (Lark) custom bot
// webhooks and Gotify push messages. A per-key cooldown deduplicates repeated
// alerts so a sustained outage does not trigger an alert storm.
package alert

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Level is the alert severity.
type Level int

const (
	// LevelInfo is informational.
	LevelInfo Level = iota
	// LevelWarning indicates a degraded condition.
	LevelWarning
	// LevelCritical indicates a failure requiring attention.
	LevelCritical
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "info"
	case LevelWarning:
		return "warning"
	case LevelCritical:
		return "critical"
	default:
		return fmt.Sprintf("level(%d)", int(l))
	}
}

// Alert is a single alert event delivered to notifiers.
type Alert struct {
	// Title is a short summary, e.g. "order-service 5xx rate high".
	Title string
	// Message is the human-readable detail.
	Message string
	// Level is the severity.
	Level Level
	// Service is the originating service name.
	Service string
	// Fields carries extra key/value context (metric values, instance IDs...).
	Fields map[string]string
	// Time is the alert occurrence time; defaults to now.
	Time time.Time
}

// WithField returns a copy of the alert with an additional field.
func (a Alert) WithField(key, value string) Alert {
	cp := a
	if cp.Fields == nil {
		cp.Fields = make(map[string]string)
	} else {
		cp.Fields = make(map[string]string, len(a.Fields))
		for k, v := range a.Fields {
			cp.Fields[k] = v
		}
	}
	cp.Fields[key] = value
	return cp
}

// normalize fills defaults (Time).
func (a Alert) normalize() Alert {
	if a.Time.IsZero() {
		a.Time = time.Now()
	}
	if a.Fields == nil {
		a.Fields = map[string]string{}
	}
	return a
}

// Notifier delivers alerts to an external channel.
type Notifier interface {
	// Name returns the notifier name for logging.
	Name() string
	// Notify sends the alert. Implementations should respect ctx cancellation.
	Notify(ctx context.Context, a Alert) error
}

// Manager fans out alerts to notifiers with per-key cooldown deduplication.
type Manager struct {
	notifiers []Notifier
	cooldown  time.Duration
	keyFunc   func(Alert) string

	mu      sync.Mutex
	lastSent map[string]time.Time
}

// Option configures a Manager.
type Option func(*Manager)

// WithCooldown sets the deduplication window per alert key (default 1h).
// Within the window, repeated alerts with the same key are dropped.
func WithCooldown(d time.Duration) Option {
	return func(m *Manager) { m.cooldown = d }
}

// WithKeyFunc customizes the deduplication key (default: service + title).
func WithKeyFunc(fn func(Alert) string) Option {
	return func(m *Manager) { m.keyFunc = fn }
}

// NewManager creates an alert manager.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		cooldown: time.Hour,
		lastSent: make(map[string]time.Time),
	}
	m.keyFunc = func(a Alert) string {
		return a.Service + "/" + a.Title
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// AddNotifier registers a delivery channel.
func (m *Manager) AddNotifier(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, n)
}

// Send delivers the alert asynchronously, subject to cooldown deduplication.
// Delivery failures are silently dropped (callers can use SendSync to observe
// errors); the cooldown only advances on successful delivery so a failing
// notifier does not swallow future alerts.
func (m *Manager) Send(ctx context.Context, a Alert) {
	a = a.normalize()

	m.mu.Lock()
	key := m.keyFunc(a)
	if !m.allowed(key, a.Time) {
		m.mu.Unlock()
		return
	}
	notifiers := make([]Notifier, len(m.notifiers))
	copy(notifiers, m.notifiers)
	m.mu.Unlock()

	go func() {
		for _, n := range notifiers {
			if err := n.Notify(ctx, a); err == nil {
				m.markSent(key, a.Time)
				return // first successful channel counts
			}
		}
	}()
}

// SendSync delivers the alert synchronously to all notifiers and returns the
// first error encountered. Cooldown still applies.
func (m *Manager) SendSync(ctx context.Context, a Alert) error {
	a = a.normalize()

	m.mu.Lock()
	key := m.keyFunc(a)
	if !m.allowed(key, a.Time) {
		m.mu.Unlock()
		return nil // deduplicated
	}
	notifiers := make([]Notifier, len(m.notifiers))
	copy(notifiers, m.notifiers)
	m.mu.Unlock()

	var firstErr error
	for _, n := range notifiers {
		if err := n.Notify(ctx, a); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		m.markSent(key, a.Time)
		return nil
	}
	return firstErr
}

// allowed reports whether the alert key may be sent now (not in cooldown).
func (m *Manager) allowed(key string, now time.Time) bool {
	if m.cooldown <= 0 {
		return true
	}
	if last, ok := m.lastSent[key]; ok && now.Sub(last) < m.cooldown {
		return false
	}
	return true
}

// markSent records the last successful delivery time for the key.
func (m *Manager) markSent(key string, t time.Time) {
	if m.cooldown <= 0 {
		return
	}
	m.mu.Lock()
	m.lastSent[key] = t
	m.mu.Unlock()
}
