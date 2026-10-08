package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMapPreservesOrder(t *testing.T) {
	out, err := Map(context.Background(), 4, []int{1, 2, 3, 4, 5}, func(_ context.Context, v int) (int, error) {
		return v * 2, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{2, 4, 6, 8, 10} {
		if out[i] != want {
			t.Fatalf("out[%d]=%d want %d", i, out[i], want)
		}
	}
}

func TestMapCancelsOnError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Map(context.Background(), 4, []int{1, 2, 3}, func(_ context.Context, v int) (int, error) {
		if v == 2 {
			return 0, boom
		}
		time.Sleep(50 * time.Millisecond)
		return v, nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v want boom", err)
	}
}

func TestSendRespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := make(chan int)
	if Send(ctx, ch, 1) {
		t.Fatal("Send should fail on cancelled context")
	}
}

func TestGoDetachedRecoversPanic(t *testing.T) {
	done := GoDetached(context.Background(), func(context.Context) error {
		panic("boom")
	})
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected panic error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSuperviseRestartsThenSucceeds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := 0
	restarts := 0
	err := Supervise(ctx, 5, time.Millisecond, func(context.Context) error {
		calls++
		if calls < 3 {
			panic("boom")
		}
		return nil
	}, func(any, error, int) { restarts++ })
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if calls != 3 || restarts != 2 {
		t.Fatalf("calls=%d restarts=%d", calls, restarts)
	}
}

func TestSuperviseGivesUp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Supervise(ctx, 1, time.Millisecond, func(context.Context) error {
		panic("always")
	}, nil)
	if err == nil {
		t.Fatal("expected error after max restarts")
	}
}

func TestSuperviseRespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Supervise(ctx, 5, time.Millisecond, func(context.Context) error {
		return errors.New("fail")
	}, nil); err == nil {
		t.Fatal("expected error")
	}
}
