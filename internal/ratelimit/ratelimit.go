// Package ratelimit provides a goroutine-safe token-bucket rate limiter
// plus a keyed registry for per-actor limits (e.g. per provider/model).
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Limiter is a token bucket: capacity burst tokens, refilled at rate tokens/sec.
type Limiter struct {
	mu       sync.Mutex
	capacity float64
	tokens   float64
	rate     float64 // tokens per second
	last     time.Time
}

// New returns a Limiter with burst capacity and refill rate per second.
// rate <= 0 means no refill (fixed budget); capacity <= 0 is normalized to 1.
func New(rate float64, capacity int) *Limiter {
	if capacity <= 0 {
		capacity = 1
	}
	if rate < 0 {
		rate = 0
	}
	return &Limiter{
		capacity: float64(capacity),
		tokens:   float64(capacity),
		rate:     rate,
		last:     time.Now(),
	}
}

// refill adds tokens accrued since last. Caller must hold mu.
func (l *Limiter) refill(now time.Time) {
	elapsed := now.Sub(l.last).Seconds()
	if elapsed <= 0 {
		return
	}
	l.tokens += elapsed * l.rate
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.last = now
}

// Allow reports whether one token is available, consuming it if so.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(time.Now())
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

// AllowN reports whether n tokens are available, consuming them if so.
func (l *Limiter) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(time.Now())
	if l.tokens < float64(n) {
		return false
	}
	l.tokens -= float64(n)
	return true
}

// Wait blocks until a token is available or ctx is done.
func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		l.refill(time.Now())
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		// Time until next token (rate > 0) else poll slowly.
		var wait time.Duration
		if l.rate > 0 {
			wait = time.Duration(float64(time.Second) * (1 - l.tokens) / l.rate)
			if wait <= 0 {
				wait = time.Millisecond
			}
		} else {
			wait = 10 * time.Millisecond
		}
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Tokens returns the currently available tokens (approximate).
func (l *Limiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill(time.Now())
	return l.tokens
}

// Registry holds named limiters (e.g. per provider).
type Registry struct {
	mu       sync.RWMutex
	limiters map[string]*Limiter
	rate     float64
	capacity int
}

// NewRegistry returns a Registry that lazily creates limiters with the
// given default rate/capacity.
func NewRegistry(rate float64, capacity int) *Registry {
	return &Registry{limiters: make(map[string]*Limiter), rate: rate, capacity: capacity}
}

// For returns the limiter for key, creating it on first use.
func (r *Registry) For(key string) *Limiter {
	r.mu.RLock()
	l, ok := r.limiters[key]
	r.mu.RUnlock()
	if ok {
		return l
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.limiters[key]; ok {
		return l
	}
	l = New(r.rate, r.capacity)
	r.limiters[key] = l
	return l
}

// Allow is a shortcut for For(key).Allow().
func (r *Registry) Allow(key string) bool { return r.For(key).Allow() }
