package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishSubscribe(t *testing.T) {
	b := NewBroker[string]()
	defer b.Shutdown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := b.Subscribe(ctx)
	require.Equal(t, 1, b.GetSubscriberCount())

	b.Publish(CreatedEvent, "hello")
	select {
	case e := <-ch:
		require.Equal(t, CreatedEvent, e.Type)
		require.Equal(t, "hello", e.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestUnsubscribeOnCancel(t *testing.T) {
	b := NewBroker[string]()
	defer b.Shutdown()
	ctx, cancel := context.WithCancel(context.Background())

	ch := b.Subscribe(ctx)
	require.Equal(t, 1, b.GetSubscriberCount())
	cancel()
	require.Eventually(t, func() bool { return b.GetSubscriberCount() == 0 }, time.Second, 10*time.Millisecond)
	_, ok := <-ch
	require.False(t, ok)
}

func TestShutdownClosesSubscribers(t *testing.T) {
	b := NewBrokerWithOptions[string](4, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := b.Subscribe(ctx)
	b.Publish(UpdatedEvent, "x")
	<-ch
	b.Shutdown()
	b.Shutdown() // idempotent
	require.Equal(t, 0, b.GetSubscriberCount())
	_, ok := <-ch
	require.False(t, ok)

	// publish after shutdown is a no-op
	b.Publish(DeletedEvent, "y")
	// subscribe after shutdown returns closed channel
	ch2 := b.Subscribe(ctx)
	_, ok = <-ch2
	require.False(t, ok)
}

func TestPublishNoSubscribers(t *testing.T) {
	b := NewBroker[int]()
	defer b.Shutdown()
	require.NotPanics(t, func() { b.Publish(CreatedEvent, 1) })
}
