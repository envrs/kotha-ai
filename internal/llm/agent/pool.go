package agent

import (
	"sync"
	"time"
)

// Pool provides reusable resources across concurrent agent sessions.
// Each resource is created lazily and released when no longer needed.
type Pool[T any] struct {
	available chan T
	created   int
	mu        sync.Mutex
	factory   func() (T, error)
	close     func(T) error
	maxSize   int
}

// NewPool returns a Pool with the given factory and max size.
func NewPool[T any](factory func() (T, error), close func(T) error, maxSize int) *Pool[T] {
	if maxSize <= 0 {
		maxSize = 4
	}
	return &Pool[T]{
		available: make(chan T, maxSize),
		factory:   factory,
		close:     close,
		maxSize:   maxSize,
	}
}

// Acquire gets a resource from the pool, creating one if necessary. Blocks
// when the pool is exhausted and a resource is in use.
func (p *Pool[T]) Acquire() (T, error) {
	select {
	case r := <-p.available:
		return r, nil
	default:
	}

	p.mu.Lock()
	if p.created < p.maxSize {
		p.created++
		p.mu.Unlock()
		r, err := p.factory()
		if err != nil {
			p.mu.Lock()
			p.created--
			p.mu.Unlock()
			var zero T
			return zero, err
		}
		return r, nil
	}
	p.mu.Unlock()

	// Pool is full; wait for a resource to be released.
	r, ok := <-p.available
	if !ok {
		var zero T
		return zero, nil
	}
	return r, nil
}

// Release returns a resource to the pool.
func (p *Pool[T]) Release(r T) {
	select {
	case p.available <- r:
	default:
		// Pool is full; close the resource instead.
		if p.close != nil {
			_ = p.close(r)
		}
	}
}

// Close closes all resources in the pool.
func (p *Pool[T]) Close() error {
	var lastErr error
	close(p.available)
	for r := range p.available {
		if p.close != nil {
			if err := p.close(r); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// Len returns the number of available resources.
func (p *Pool[T]) Len() int {
	return len(p.available)
}

var _ = time.Now