// Package health provides a lightweight component health registry:
// checks register by name, run with a context (timeout/cancel aware),
// and aggregate into an overall status without one slow check blocking others.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/kothagpt/kotha/internal/concurrency"
)

// Status is the outcome of a single check.
type Status string

const (
	StatusUp       Status = "up"
	StatusDown     Status = "down"
	StatusUnknown  Status = "unknown"
	StatusDegraded Status = "degraded"
)

// CheckFunc probes one component. It must respect ctx cancellation.
type CheckFunc func(ctx context.Context) error

// Result is the outcome of one named check.
type Result struct {
	Name     string
	Status   Status
	Latency  time.Duration
	Error    error
	Checked  time.Time
}

// Checker holds the registered checks.
type Checker struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

// New returns an empty Checker.
func New() *Checker {
	return &Checker{checks: make(map[string]CheckFunc)}
}

// Register adds or replaces the check for name.
func (c *Checker) Register(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// Unregister removes the check for name.
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.checks, name)
}

// Names returns the registered check names.
func (c *Checker) Names() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.checks))
	for n := range c.checks {
		names = append(names, n)
	}
	return names
}

// CheckOne runs a single named check with ctx.
func (c *Checker) CheckOne(ctx context.Context, name string) Result {
	c.mu.RLock()
	fn, ok := c.checks[name]
	c.mu.RUnlock()
	if !ok {
		return Result{Name: name, Status: StatusUnknown, Checked: time.Now()}
	}
	start := time.Now()
	err := fn(ctx)
	r := Result{Name: name, Latency: time.Since(start), Checked: time.Now()}
	if err != nil {
		r.Status = StatusDown
		r.Error = err
	} else {
		r.Status = StatusUp
	}
	return r
}

// Report aggregates Check results.
type Report struct {
	Results []Result
	Checked time.Time
}

// Overall folds results: any down => down, empty => unknown, else up.
func (r Report) Overall() Status {
	if len(r.Results) == 0 {
		return StatusUnknown
	}
	for _, res := range r.Results {
		if res.Status == StatusDown {
			return StatusDown
		}
	}
	return StatusUp
}

// Check runs all registered checks concurrently (bounded) and returns a Report.
// A single slow check cannot block the others beyond ctx cancellation.
func (c *Checker) Check(ctx context.Context) Report {
	c.mu.RLock()
	names := make([]string, 0, len(c.checks))
	fns := make([]CheckFunc, 0, len(c.checks))
	for n, fn := range c.checks {
		names = append(names, n)
		fns = append(fns, fn)
	}
	c.mu.RUnlock()

	results := make([]Result, len(names))
	gr := concurrency.WithGroup(ctx, 8)
	for i := range names {
		gr.Go(func(ctx context.Context) error {
			start := time.Now()
			err := fns[i](ctx)
			r := Result{Name: names[i], Latency: time.Since(start), Checked: time.Now()}
			if err != nil {
				r.Status = StatusDown
				r.Error = err
			} else {
				r.Status = StatusUp
			}
			results[i] = r
			return nil
		})
	}
	_ = gr.Wait()
	return Report{Results: results, Checked: time.Now()}
}
