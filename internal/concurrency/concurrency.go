// Package concurrency provides small structured-concurrency primitives:
// supervised goroutines bound to a context, errgroup-based fan-out with
// bounded parallelism, and context-aware channel sends.
//
// Rules enforced by these helpers:
//   - every spawned goroutine is owned by a Group or tied to a context,
//   - blocking sends always select on ctx.Done(),
//   - panics are recovered and surfaced as errors, never silent deaths.
package concurrency

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"golang.org/x/sync/errgroup"
)

// Group is a supervised errgroup bound to a parent context.
// Canceling the parent cancels all members; the first error cancels the rest.
type Group struct {
	g   *errgroup.Group
	ctx context.Context
}

// WithGroup derives a Group (and child context) from parent.
// limit <= 0 means unbounded; limit > 0 bounds concurrent members.
func WithGroup(parent context.Context, limit int) *Group {
	g, ctx := errgroup.WithContext(parent)
	if limit > 0 {
		g.SetLimit(limit)
	}
	return &Group{g: g, ctx: ctx}
}

// Go spawns fn supervised by the group with panic recovery.
func (gr *Group) Go(fn func(ctx context.Context) error) {
	gr.g.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
			}
		}()
		return fn(gr.ctx)
	})
}

// Wait blocks until all members complete, returning the first error.
func (gr *Group) Wait() error { return gr.g.Wait() }

// GoDetached starts a supervised goroutine owned by ctx with panic recovery
// and returns a channel carrying its error. The goroutine always terminates
// when ctx is done; callers should not leak it.
func GoDetached(ctx context.Context, fn func(ctx context.Context) error) <-chan error {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v\n%s", r, debug.Stack())
				return
			}
		}()
		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return
		default:
		}
		done <- fn(ctx)
	}()
	return done
}

// Send delivers v to ch unless ctx is done. It reports whether the send
// succeeded, so producers never block forever on an abandoned consumer.
func Send[T any](ctx context.Context, ch chan<- T, v T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- v:
		return true
	}
}

// Map runs fn over items with at most limit concurrent workers, preserving
// order. The first error cancels the remaining work.
func Map[T, R any](ctx context.Context, limit int, items []T, fn func(ctx context.Context, item T) (R, error)) ([]R, error) {
	out := make([]R, len(items))
	gr := WithGroup(ctx, limit)
	for i, item := range items {
		gr.Go(func(ctx context.Context) error {
			r, err := fn(ctx, item)
			if err != nil {
				return err
			}
			out[i] = r
			return nil
		})
	}
	if err := gr.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// Supervise runs fn in a restart loop until ctx is done, fn returns nil, or
// maxRestarts is exceeded. Panics are recovered and count as failures.
// Backoff grows linearly (backoff * attempt, capped at 30s) between restarts.
// onPanic, if non-nil, observes each failure for logging/recovery hooks.
//
// Use for long-lived goroutines (subscriptions, message pumps) so a single
// panic degrades to a logged restart instead of a permanently dead loop.
func Supervise(ctx context.Context, maxRestarts int, backoff time.Duration, fn func(ctx context.Context) error, onPanic func(recovered any, err error, attempt int)) error {
	var attempt int
	for {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
				}
			}()
			return fn(ctx)
		}()
		if err == nil || ctx.Err() != nil {
			return err
		}
		attempt++
		if onPanic != nil {
			onPanic(nil, err, attempt)
		}
		if maxRestarts >= 0 && attempt > maxRestarts {
			return err
		}
		wait := backoff * time.Duration(attempt)
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		if wait <= 0 {
			wait = 100 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}
