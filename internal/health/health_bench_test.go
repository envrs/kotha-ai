package health

import (
	"context"
	"testing"
)

func BenchmarkRegister(b *testing.B) {
	c := New()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Register("bench", func(context.Context) error { return nil })
	}
}

func BenchmarkCheckOneFast(b *testing.B) {
	c := New()
	c.Register("fast", func(context.Context) error { return nil })
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.CheckOne(ctx, "fast")
	}
}

func BenchmarkCheck10Concurrent(b *testing.B) {
	// Distinct names to avoid overwrite collapsing to 1 check.
	c2 := New()
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	for _, n := range names {
		c2.Register(n, func(context.Context) error { return nil })
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c2.Check(ctx)
	}
}

func BenchmarkOverall(b *testing.B) {
	rep := Report{Results: []Result{
		{Name: "a", Status: StatusUp},
		{Name: "b", Status: StatusUp},
	}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rep.Overall()
	}
}
