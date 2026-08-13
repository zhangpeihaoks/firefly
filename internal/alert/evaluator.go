package alert

import (
	"context"
	"strconv"
	"time"
)

// Rule is a self-contained alert evaluation rule. Implementations return the
// alert to raise and whether it triggered; the Manager's cooldown provides
// deduplication so a sustained breach does not cause an alert storm.
type Rule interface {
	// Name is the rule identifier (also the alert title).
	Name() string
	// Evaluate inspects current state and returns the alert to send, plus
	// whether it should be raised.
	Evaluate(ctx context.Context) (Alert, bool)
}

// Comparison is the threshold operator.
type Comparison string

const (
	OpGreaterThanOrEqual Comparison = ">="
	OpGreaterThan        Comparison = ">"
	OpLessThan           Comparison = "<"
	OpLessThanOrEqual    Comparison = "<="
)

// ThresholdRule raises an alert when a metric breaches a threshold.
// It is the built-in implementation of Rule; custom rules can implement the
// interface directly.
type ThresholdRule struct {
	name      string
	metric    func() float64
	op        Comparison
	threshold float64
	level     Level
	message   func() string
	service   string
}

// ThresholdOption configures a ThresholdRule.
type ThresholdOption func(*ThresholdRule)

// WithComparison sets the operator (default ">=").
func WithComparison(op Comparison) ThresholdOption {
	return func(r *ThresholdRule) { r.op = op }
}

// WithLevel sets the alert severity (default LevelWarning).
func WithLevel(l Level) ThresholdOption {
	return func(r *ThresholdRule) { r.level = l }
}

// WithMessage overrides the generated alert message.
func WithMessage(fn func() string) ThresholdOption {
	return func(r *ThresholdRule) { r.message = fn }
}

// WithService tags the alert with a service name.
func WithService(s string) ThresholdOption {
	return func(r *ThresholdRule) { r.service = s }
}

// NewThresholdRule creates a threshold rule around a metric reader.
//
//	rule := alert.NewThresholdRule("5xx_rate", func() float64 { return fivexxRate() }, 0.05,
//	    alert.WithComparison(alert.OpGreaterThanOrEqual),
//	    alert.WithLevel(alert.LevelCritical),
//	)
func NewThresholdRule(name string, metric func() float64, threshold float64, opts ...ThresholdOption) *ThresholdRule {
	r := &ThresholdRule{
		name:      name,
		metric:    metric,
		op:        OpGreaterThanOrEqual,
		threshold: threshold,
		level:     LevelWarning,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Name implements Rule.
func (r *ThresholdRule) Name() string { return r.name }

// Evaluate implements Rule: it returns the alert when the metric breaches
// the threshold.
func (r *ThresholdRule) Evaluate(_ context.Context) (Alert, bool) {
	if r.metric == nil {
		return Alert{}, false
	}
	value := r.metric()
	if !breach(r.op, value, r.threshold) {
		return Alert{}, false
	}

	msg := ""
	if r.message != nil {
		msg = r.message()
	}
	if msg == "" {
		msg = "阈值触发"
	}

	a := Alert{
		Title:   r.name,
		Message: msg,
		Level:   r.level,
		Service: r.service,
	}.WithField("value", strconv.FormatFloat(value, 'f', -1, 64)).
		WithField("threshold", strconv.FormatFloat(r.threshold, 'f', -1, 64))
	return a, true
}

// breach applies the comparison operator.
func breach(op Comparison, value, threshold float64) bool {
	switch op {
	case OpGreaterThan:
		return value > threshold
	case OpLessThan:
		return value < threshold
	case OpLessThanOrEqual:
		return value <= threshold
	default: // >=
		return value >= threshold
	}
}

// ---------------------------------------------------------------------------
// Evaluator
// ---------------------------------------------------------------------------

// Evaluator periodically evaluates rules and raises alerts through a Manager.
type Evaluator struct {
	manager  *Manager
	rules    []Rule
	interval time.Duration
	stopCh   chan struct{}
}

// EvaluatorOption configures an Evaluator.
type EvaluatorOption func(*Evaluator)

// WithInterval sets the evaluation period (default 30s).
func WithInterval(d time.Duration) EvaluatorOption {
	return func(e *Evaluator) { e.interval = d }
}

// NewEvaluator creates an evaluator that sends triggered alerts to mgr.
func NewEvaluator(mgr *Manager, rules []Rule, opts ...EvaluatorOption) *Evaluator {
	e := &Evaluator{
		manager:  mgr,
		rules:    rules,
		interval: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Start begins periodic evaluation until ctx is cancelled or Stop is called.
func (e *Evaluator) Start(ctx context.Context) {
	e.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(e.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.evaluate(ctx)
			}
		}
	}()
}

// Stop halts evaluation.
func (e *Evaluator) Stop() {
	if e.stopCh != nil {
		close(e.stopCh)
		e.stopCh = nil
	}
}

// evaluate checks every rule and sends triggered alerts (deduplicated by the
// manager's cooldown).
func (e *Evaluator) evaluate(ctx context.Context) {
	for _, r := range e.rules {
		if a, triggered := r.Evaluate(ctx); triggered {
			e.manager.Send(ctx, a)
		}
	}
}
