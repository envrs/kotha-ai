package route

import (
	"context"
	"time"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Transport moves protocol envelopes to an endpoint.
// Implementations: in-memory (tests), framed conn, HTTP.
type Transport interface {
	RoundTrip(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// Endpoint identifies where a request is routed.
type Endpoint struct {
	Name     string        `json:"name"`
	Provider string        `json:"provider"`
	Model    string        `json:"model"`
	URL      string        `json:"url,omitempty"`
	Auth     Credentials   `json:"-"`
	Timeout  time.Duration `json:"timeout,omitempty"`
}

func (e Endpoint) Validate() error {
	if e.Model == "" {
		return schema.NewError(schema.CodeInvalidRequest, schema.ErrInvalidRequest)
	}
	return nil
}

func (e Endpoint) timeoutOr(d time.Duration) time.Duration {
	if e.Timeout > 0 {
		return e.Timeout
	}
	return d
}
