// Package observability provides lightweight metrics and tracing helpers
// that can be swapped for a full OpenTelemetry backend.
package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kothagpt/kotha/internal/logging"
)

// Counter is a monotonically increasing metric.
type Counter struct {
	mu    sync.Mutex
	value int64
	name  string
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Add(n int64) {
	c.mu.Lock()
	c.value += n
	c.mu.Unlock()
}

func (c *Counter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Histogram records the distribution of values.
type Histogram struct {
	mu    sync.Mutex
	sum   float64
	count int64
	name  string
}

func NewHistogram(name string) *Histogram {
	return &Histogram{name: name}
}

func (h *Histogram) Record(v float64) {
	h.mu.Lock()
	h.sum += v
	h.count++
	h.mu.Unlock()
}

func (h *Histogram) Avg() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// Tracer is a minimal tracing interface used by the agent loop.
type Tracer interface {
	Start(ctx context.Context, name string) (Span, error)
}

// Span represents a unit of work.
type Span interface {
	SetAttribute(key, value string)
	End()
}

type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, name string) (Span, error) {
	return noopSpan{}, nil
}

type noopSpan struct{}

func (noopSpan) SetAttribute(key, value string) {}
func (noopSpan) End()                              {}

// NewNoopTracer returns a tracer that does nothing.
func NewNoopTracer() Tracer { return noopTracer{} }

// loggingTracer emits spans as log lines.
type loggingTracer struct{}

func (loggingTracer) Start(ctx context.Context, name string) (Span, error) {
	logging.Debug("trace: span started", "name", name)
	return &loggingSpan{name: name, start: time.Now()}, nil
}

type loggingSpan struct {
	name  string
	start time.Time
}

func (s *loggingSpan) SetAttribute(key, value string) {
	logging.Debug("trace: span attribute", "name", s.name, key, value)
}

func (s *loggingSpan) End() {
	logging.Debug("trace: span ended", "name", s.name, "duration", time.Since(s.start))
}

// NewLoggingTracer returns a tracer that emits spans as debug log lines.
func NewLoggingTracer() Tracer { return loggingTracer{} }

// Meter exposes the counters and histograms used by the agent loop.
type Meter struct {
	ToolCalls    *Counter
	ToolErrors   *Counter
	AgentRuns    *Counter
	Latency      *Histogram
}

// NewMeter constructs a Meter with fresh counters and histograms.
func NewMeter() *Meter {
	return &Meter{
		ToolCalls:  NewCounter("tool_calls"),
		ToolErrors: NewCounter("tool_errors"),
		AgentRuns:  NewCounter("agent_runs"),
		Latency:    NewHistogram("agent_latency_ms"),
	}
}

// Report prints a summary to the logger.
func (m *Meter) Report() {
	logging.Info("metrics report",
		"tool_calls", m.ToolCalls.Value(),
		"tool_errors", m.ToolErrors.Value(),
		"agent_runs", m.AgentRuns.Value(),
		"avg_latency_ms", fmt.Sprintf("%.2f", m.Latency.Avg()),
	)
}

// contextKey is used to carry a Meter through request contexts.
type contextKey string

const meterKey contextKey = "meter"

// WithMeter attaches a Meter to the context.
func WithMeter(ctx context.Context, m *Meter) context.Context {
	return context.WithValue(ctx, meterKey, m)
}

// FromContext returns the Meter stored in the context, or nil.
func FromContext(ctx context.Context) *Meter {
	if v := ctx.Value(meterKey); v != nil {
		return v.(*Meter)
	}
	return nil
}