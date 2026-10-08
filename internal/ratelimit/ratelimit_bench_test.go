package ratelimit

import (
	"testing"
)

func BenchmarkAllowUncontended(b *testing.B) {
	l := New(1e9, 1e9) // effectively unlimited
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}

func BenchmarkAllowContended(b *testing.B) {
	l := New(1e9, 1e9)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = l.Allow()
		}
	})
}

func BenchmarkAllowDenied(b *testing.B) {
	l := New(0, 1)
	_ = l.Allow() // drain budget; every call now takes the deny path
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.Allow()
	}
}

func BenchmarkRegistryFor(b *testing.B) {
	r := NewRegistry(1e9, 100)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = r.Allow("openai")
		}
	})
}
