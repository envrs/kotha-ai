package pubsub

import (
	"context"
	"testing"
)

// Event system benchmarks: the pubsub Broker is the process event bus
// (sessions, messages, permissions, agent events -> TUI).

func BenchmarkPublishNoSubscribers(b *testing.B) {
	br := NewBroker[string]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Publish(CreatedEvent, "payload")
	}
}

func BenchmarkPublishSingleSubscriber(b *testing.B) {
	br := NewBroker[string]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := br.Subscribe(ctx)
	// Drain so the buffer never fills and Publish stays on the fast path.
	go func() {
		for range sub {
		}
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Publish(CreatedEvent, "payload")
	}
}

func BenchmarkPublishManySubscribers(b *testing.B) {
	for _, n := range []int{10, 100} {
		b.Run(map[int]string{10: "10subs", 100: "100subs"}[n], func(b *testing.B) {
			br := NewBroker[string]()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			for i := 0; i < n; i++ {
				sub := br.Subscribe(ctx)
				go func() {
					for range sub {
					}
				}()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				br.Publish(UpdatedEvent, "payload")
			}
		})
	}
}

func BenchmarkSubscribeUnsubscribe(b *testing.B) {
	br := NewBroker[string]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		_ = br.Subscribe(ctx)
		cancel()
	}
}
