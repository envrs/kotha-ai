package route

import (
	"context"
	"errors"

	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Backend is a chat+stream client. Client satisfies it, so backends are
// usually plain Clients bound to an endpoint + transport.
type Backend interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...schema.Option) (schema.Completion, error)
	Stream(ctx context.Context, messages []schema.Message, opts ...schema.Option) (<-chan schema.StreamEvent, error)
}

// FallbackClient routes requests across an ordered list of backends. It
// tries each backend in turn and falls through only on errors that are
// likely to succeed elsewhere (rate limits, auth, provider outages).
// Request-shaped failures (invalid request, context too large, cancelled)
// are terminal because every backend would reject them identically.
//
// Stream fallback applies to stream establishment only; once a stream is
// open, mid-stream errors are forwarded to the caller unchanged.
type FallbackClient struct {
	Backends []Backend
}

// NewFallbackClient builds a FallbackClient from ordered backends.
func NewFallbackClient(backends ...Backend) FallbackClient {
	return FallbackClient{Backends: backends}
}

// Fallbackable reports whether err should cause a switch to the next
// backend rather than failing the request.
func Fallbackable(err error) bool {
	if err == nil {
		return false
	}
	if schema.IsRetryable(err) {
		return true
	}
	switch schema.CodeOf(err) {
	case schema.CodeRateLimited, schema.CodeProvider, schema.CodeAuth, schema.CodeBusy, schema.CodeUnknown:
		return true
	}
	return false
}

func (f FallbackClient) Chat(ctx context.Context, messages []schema.Message, opts ...schema.Option) (schema.Completion, error) {
	if len(f.Backends) == 0 {
		return schema.Completion{}, schema.NewError(schema.CodeInvalidRequest, errors.New("route: no backends configured"))
	}
	var last error
	for i, b := range f.Backends {
		if err := ctx.Err(); err != nil {
			return schema.Completion{}, schema.NewError(schema.CodeCancelled, err)
		}
		comp, err := b.Chat(ctx, messages, opts...)
		if err == nil {
			return comp, nil
		}
		last = err
		if !Fallbackable(err) || i == len(f.Backends)-1 {
			return schema.Completion{}, err
		}
	}
	return schema.Completion{}, last
}

func (f FallbackClient) Stream(ctx context.Context, messages []schema.Message, opts ...schema.Option) (<-chan schema.StreamEvent, error) {
	if len(f.Backends) == 0 {
		return nil, schema.NewError(schema.CodeInvalidRequest, errors.New("route: no backends configured"))
	}
	var last error
	for i, b := range f.Backends {
		if err := ctx.Err(); err != nil {
			return nil, schema.NewError(schema.CodeCancelled, err)
		}
		ch, err := b.Stream(ctx, messages, opts...)
		if err == nil {
			return ch, nil
		}
		last = err
		if !Fallbackable(err) || i == len(f.Backends)-1 {
			return nil, err
		}
	}
	return nil, last
}

// SelectCheapest returns the cheapest model ID among ids by list price
// (input + output dollars per 1M tokens). Unknown model IDs are skipped;
// returns "" when no id resolves to a known model.
func SelectCheapest(ids []string) string {
	bestID := ""
	bestPrice := 0.0
	for _, id := range ids {
		m, ok := models.SupportedModels[models.ModelID(id)]
		if !ok {
			continue
		}
		price := m.CostPer1MIn + m.CostPer1MOut
		if bestID == "" || price < bestPrice {
			bestID, bestPrice = id, price
		}
	}
	return bestID
}
