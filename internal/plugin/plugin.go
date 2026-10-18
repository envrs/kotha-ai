package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kothagpt/kotha/internal/message"
)

// PromptRequest is the admission-time view of a user prompt, handed to
// interceptors before the prompt is persisted or sent to the LLM.
type PromptRequest struct {
	SessionID   string
	Content     string
	Attachments []message.Attachment
}

// PromptInterceptor sees every user prompt before admission. It may inspect
// the request, rewrite Content or Attachments, or reject it by returning a
// *BlockedError (or any error, treated as a rejection).
type PromptInterceptor interface {
	// Name identifies the interceptor (used in logs and block reasons).
	Name() string
	// InterceptPrompt is called once per user prompt, in registration order.
	InterceptPrompt(ctx context.Context, req *PromptRequest) error
}

// BlockedError rejects a prompt with a human-readable reason. Wrap one when
// an interceptor denies admission.
type BlockedError struct {
	Interceptor string
	Reason      string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("prompt blocked by %q: %s", e.Interceptor, e.Reason)
}

// Block is a convenience constructor for interceptors.
func Block(interceptor, reason string) *BlockedError {
	return &BlockedError{Interceptor: interceptor, Reason: reason}
}

// funcInterceptor adapts a bare function to PromptInterceptor.
type funcInterceptor struct {
	name string
	fn   func(ctx context.Context, req *PromptRequest) error
}

func (f *funcInterceptor) Name() string { return f.name }

func (f *funcInterceptor) InterceptPrompt(ctx context.Context, req *PromptRequest) error {
	return f.fn(ctx, req)
}

// Func turns a function into a PromptInterceptor so callers can register
// hooks without declaring a new type:
//
//	chain.Use(plugin.Func("deny-secrets", func(ctx context.Context,
//	req *plugin.PromptRequest) error { ... }))
func Func(name string, fn func(ctx context.Context, req *PromptRequest) error) PromptInterceptor {
	return &funcInterceptor{name: name, fn: fn}
}

// Chain runs registered interceptors sequentially, in registration order.
// It is safe for concurrent registration and admission.
type Chain struct {
	mu           sync.RWMutex
	interceptors []PromptInterceptor
}

func NewChain() *Chain { return &Chain{} }

// Use appends an interceptor to the chain. Nil interceptors are ignored.
func (c *Chain) Use(i PromptInterceptor) {
	if i == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, i)
}

// Len reports how many interceptors are currently registered.
func (c *Chain) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.interceptors)
}

// Names returns the registered interceptors in run order.
func (c *Chain) Names() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.interceptors))
	for _, i := range c.interceptors {
		names = append(names, i.Name())
	}
	return names
}

// Intercept runs every interceptor in registration order, stopping at the
// first rejection. Interceptors may mutate req (Content, Attachments); the
// mutated request is what the agent admits.
func (c *Chain) Intercept(ctx context.Context, req *PromptRequest) error {
	c.mu.RLock()
	interceptors := make([]PromptInterceptor, len(c.interceptors))
	copy(interceptors, c.interceptors)
	c.mu.RUnlock()

	for _, i := range interceptors {
		if err := i.InterceptPrompt(ctx, req); err != nil {
			var blocked *BlockedError
			if errors.As(err, &blocked) {
				return err
			}
			return &BlockedError{Interceptor: i.Name(), Reason: err.Error()}
		}
	}
	return nil
}
