package route

import (
	"context"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Client is the high-level entry point: builds protocol requests from
// schema types and runs them through the Executor.
type Client struct {
	Endpoint Endpoint
	Executor Executor
}

func NewClient(ep Endpoint, t Transport) Client {
	return Client{Endpoint: ep, Executor: Executor{Transport: t}}
}

func (c Client) Chat(ctx context.Context, messages []schema.Message, opts ...schema.Option) (schema.Completion, error) {
	o := schema.NewRequestOptions(opts...)
	if o.Model == "" {
		o.Model = c.Endpoint.Model
	}
	req := ChatRequest{
		RequestID: schema.NewRequestID(),
		Kind:      KindChat,
		Endpoint:  c.Endpoint,
		Messages:  messages,
		Options:   o,
	}
	resp, err := c.Executor.Do(ctx, req)
	if err != nil {
		return schema.Completion{}, err
	}
	return resp.Completion, nil
}

func (c Client) Stream(ctx context.Context, messages []schema.Message, opts ...schema.Option) (<-chan schema.StreamEvent, error) {
	o := schema.NewRequestOptions(opts...)
	if o.Model == "" {
		o.Model = c.Endpoint.Model
	}
	req := ChatRequest{
		RequestID: schema.NewRequestID(),
		Kind:      KindStream,
		Endpoint:  c.Endpoint,
		Messages:  messages,
		Options:   o,
	}
	chunks, err := c.Executor.DoStream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan schema.StreamEvent, 16)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case ch, ok := <-chunks:
				if !ok {
					return
				}
				ev := ch.Event
				ev.RequestID = ch.RequestID
				select {
				case <-ctx.Done():
					return
				case out <- ev:
				}
				if ch.Final || ev.IsTerminal() {
					return
				}
			}
		}
	}()
	return out, nil
}
