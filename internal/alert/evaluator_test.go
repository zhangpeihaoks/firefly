package alert

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestThresholdRule_Breaches(t *testing.T) {
	value := 0.08
	rule := NewThresholdRule("5xx_rate", func() float64 { return value }, 0.05,
		WithLevel(LevelCritical),
		WithService("order-service"),
	)

	a, triggered := rule.Evaluate(context.Background())
	if !triggered {
		t.Fatal("expected rule to trigger")
	}
	if a.Title != "5xx_rate" {
		t.Errorf("title = %s, want 5xx_rate", a.Title)
	}
	if a.Level != LevelCritical {
		t.Errorf("level = %v, want critical", a.Level)
	}
	if a.Service != "order-service" {
		t.Errorf("service = %s", a.Service)
	}
	if a.Fields["value"] != "0.08" || a.Fields["threshold"] != "0.05" {
		t.Errorf("unexpected fields: %v", a.Fields)
	}
}

func TestThresholdRule_NotBreach(t *testing.T) {
	rule := NewThresholdRule("cpu", func() float64 { return 0.3 }, 0.9)
	if _, triggered := rule.Evaluate(context.Background()); triggered {
		t.Fatal("expected rule NOT to trigger below threshold")
	}
}

func TestThresholdRule_Operators(t *testing.T) {
	tests := []struct {
		name      string
		op        Comparison
		value     float64
		threshold float64
		want      bool
	}{
		{"ge above", OpGreaterThanOrEqual, 5, 5, true},
		{"gt equal", OpGreaterThan, 5, 5, false},
		{"lt below", OpLessThan, 3, 5, true},
		{"lt equal", OpLessThan, 5, 5, false},
		{"le equal", OpLessThanOrEqual, 5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewThresholdRule("m", func() float64 { return tt.value }, tt.threshold,
				WithComparison(tt.op))
			_, got := rule.Evaluate(context.Background())
			if got != tt.want {
				t.Errorf("breach = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThresholdRule_CustomMessage(t *testing.T) {
	rule := NewThresholdRule("latency", func() float64 { return 3.0 }, 1.0,
		WithMessage(func() string { return "p99 latency exploded" }),
	)

	a, triggered := rule.Evaluate(context.Background())
	if !triggered {
		t.Fatal("expected trigger")
	}
	if a.Message != "p99 latency exploded" {
		t.Errorf("message = %s", a.Message)
	}
}

// customRule is a user-defined Rule implementation (interface + impl pattern).
type customRule struct{}

func (customRule) Name() string { return "custom-rule" }

func (customRule) Evaluate(ctx context.Context) (Alert, bool) {
	select {
	case <-ctx.Done():
		return Alert{}, false
	default:
	}
	return Alert{Title: "custom", Message: "custom logic", Level: LevelInfo}, true
}

func TestEvaluator_WithCustomRule(t *testing.T) {
	delivered := make(chan Alert, 1)
	mgr := NewManager()
	mgr.AddNotifier(notifierFunc(func(ctx context.Context, a Alert) error {
		delivered <- a
		return nil
	}))

	ev := NewEvaluator(mgr, []Rule{customRule{}}, WithInterval(10*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	ev.Start(ctx)
	defer func() { cancel(); ev.Stop() }()

	select {
	case a := <-delivered:
		if a.Title != "custom" {
			t.Errorf("alert title = %s, want custom", a.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("custom rule alert was not delivered")
	}
}

func TestThresholdRule_DefaultLevel(t *testing.T) {
	rule := NewThresholdRule("m", func() float64 { return 1 }, 0)
	a, _ := rule.Evaluate(context.Background())
	if a.Level != LevelWarning {
		t.Errorf("default level = %v, want warning", a.Level)
	}
	if !strings.Contains(a.Message, "阈值触发") {
		t.Errorf("default message = %q", a.Message)
	}
}

// notifierFunc adapts a function to the Notifier interface.
type notifierFunc func(ctx context.Context, a Alert) error

func (f notifierFunc) Name() string { return "func" }

func (f notifierFunc) Notify(ctx context.Context, a Alert) error { return f(ctx, a) }
