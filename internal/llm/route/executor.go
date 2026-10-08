package route

import (
	"context"
	"time"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Executor runs requests through a Transport with timeout + retry on
// retryable schema errors.
type Executor struct {
	Transport  Transport
	Timeout    time.Duration
	MaxRetries int
	Backoff    func(attempt int) time.Duration
}

func (e Executor) timeout() time.Duration {
	if e.Timeout > 0 {
		return e.Timeout
	}
	return schema.DefaultTimeout
}

func (e Executor) maxRetries() int {
	if e.MaxRetries > 0 {
		return e.MaxRetries
	}
	return schema.DefaultMaxRetries
}

func (e Executor) backoff(attempt int) time.Duration {
	if e.Backoff != nil {
		return e.Backoff(attempt)
	}
	d := 250 * time.Millisecond * time.Duration(1<<attempt)
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}

func (e Executor) Do(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if err := req.Endpoint.Validate(); err != nil {
		return ChatResponse{}, err
	}
	if err := req.Validate(); err != nil {
		return ChatResponse{}, err
	}
	timeout := e.timeout()
	if ep := req.Endpoint.timeoutOr(0); ep > 0 {
		timeout = ep
	}
	var last error
	for attempt := 0; attempt <= e.maxRetries(); attempt++ {
		inner, cancel := context.WithTimeout(ctx, timeout)
		resp, err := e.Transport.RoundTrip(inner, req)
		cancel()
		if err == nil {
			return resp, nil
		}
		last = err
		if ctx.Err() != nil || !schema.IsRetryable(err) {
			return ChatResponse{}, err
		}
		select {
		case <-ctx.Done():
			return ChatResponse{}, ctx.Err()
		case <-time.After(e.backoff(attempt)):
		}
	}
	return ChatResponse{}, schema.NewError(schema.CodeMaxRetries, last)
}

func (e Executor) DoStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	if err := req.Endpoint.Validate(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.Kind = KindStream
	return e.Transport.Stream(ctx, req)
}
