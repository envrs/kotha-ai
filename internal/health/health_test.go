package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckAggregates(t *testing.T) {
	c := New()
	c.Register("ok", func(context.Context) error { return nil })
	c.Register("bad", func(context.Context) error { return errors.New("down") })
	rep := c.Check(context.Background())
	if len(rep.Results) != 2 {
		t.Fatalf("results=%d", len(rep.Results))
	}
	if rep.Overall() != StatusDown {
		t.Fatalf("overall=%s", rep.Overall())
	}
	if got := c.CheckOne(context.Background(), "missing"); got.Status != StatusUnknown {
		t.Fatalf("missing=%s", got.Status)
	}
}

func TestCheckRespectsCancel(t *testing.T) {
	c := New()
	c.Register("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			return nil
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	rep := c.Check(ctx)
	if rep.Overall() != StatusDown {
		t.Fatalf("overall=%s", rep.Overall())
	}
}
