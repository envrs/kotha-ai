package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounter(t *testing.T) {
	c := NewCounter("tool_calls")
	require.Equal(t, int64(0), c.Value())
	c.Add(5)
	require.Equal(t, int64(5), c.Value())
	c.Add(3)
	require.Equal(t, int64(8), c.Value())
}

func TestHistogram(t *testing.T) {
	h := NewHistogram("latency")
	require.Equal(t, 0.0, h.Avg())
	h.Record(100)
	h.Record(200)
	require.InDelta(t, 150.0, h.Avg(), 0.001)
}

func TestMeterReport(t *testing.T) {
	m := NewMeter()
	m.ToolCalls.Add(10)
	m.ToolErrors.Add(2)
	m.AgentRuns.Add(3)
	m.Latency.Record(50)
	require.Equal(t, int64(10), m.ToolCalls.Value())
	require.Equal(t, int64(2), m.ToolErrors.Value())
	require.Equal(t, int64(3), m.AgentRuns.Value())
	m.Report() // should not panic
}

func TestMeterContext(t *testing.T) {
	m := NewMeter()
	ctx := WithMeter(context.Background(), m)
	got := FromContext(ctx)
	require.Same(t, m, got)
	require.Nil(t, FromContext(context.Background()))
}

func TestTracer(t *testing.T) {
	tr := NewNoopTracer()
	span, err := tr.Start(context.Background(), "test")
	require.NoError(t, err)
	span.SetAttribute("key", "value")
	span.End()
}