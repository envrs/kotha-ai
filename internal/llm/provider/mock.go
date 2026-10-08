package provider

import (
	"context"
	"sync"

	"github.com/kothagpt/kotha/internal/concurrency"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
)

// MockClient is a scriptable ProviderClient for tests.
// Queue responses with WithMockResponses / Enqueue; optionally inject errors.
type MockClient ProviderClient

type mockClient struct {
	mu        sync.Mutex
	responses []*ProviderResponse
	err       error
	streamErr error

	// Captured calls for assertions.
	Calls []MockCall
}

type MockCall struct {
	Messages []message.Message
	Tools    []tools.BaseTool
}

type MockOption func(*mockClient)

func WithMockResponses(responses ...*ProviderResponse) MockOption {
	return func(m *mockClient) {
		m.responses = append(m.responses, responses...)
	}
}

func WithMockError(err error) MockOption {
	return func(m *mockClient) {
		m.err = err
	}
}

func WithMockStreamError(err error) MockOption {
	return func(m *mockClient) {
		m.streamErr = err
	}
}

func newMockClient(opts providerClientOptions) MockClient {
	m := &mockClient{}
	for _, o := range opts.mockOptions {
		o(m)
	}
	return m
}

// Enqueue adds a response to the queue at runtime (safe for concurrent use).
func (m *mockClient) Enqueue(r *ProviderResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = append(m.responses, r)
}

func (m *mockClient) next() (*ProviderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if len(m.responses) == 0 {
		return &ProviderResponse{FinishReason: message.FinishReasonEndTurn}, nil
	}
	r := m.responses[0]
	m.responses = m.responses[1:]
	return r, nil
}

func (m *mockClient) record(messages []message.Message, tools []tools.BaseTool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, MockCall{Messages: messages, Tools: tools})
}

func (m *mockClient) send(_ context.Context, messages []message.Message, tools []tools.BaseTool) (*ProviderResponse, error) {
	m.record(messages, tools)
	return m.next()
}

func (m *mockClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	ch := make(chan ProviderEvent, 8)
	go func() {
		defer close(ch)
		m.record(messages, tools)
		m.mu.Lock()
		streamErr := m.streamErr
		m.mu.Unlock()
		if streamErr != nil {
			concurrency.Send(ctx, ch, ProviderEvent{Type: EventError, Error: streamErr})
			return
		}
		resp, err := m.next()
		if err != nil {
			concurrency.Send(ctx, ch, ProviderEvent{Type: EventError, Error: err})
			return
		}
		if resp.Content != "" {
			concurrency.Send(ctx, ch, ProviderEvent{Type: EventContentDelta, Content: resp.Content})
		}
		for _, tc := range resp.ToolCalls {
			tcCopy := tc
			concurrency.Send(ctx, ch, ProviderEvent{Type: EventToolUseStart, ToolCall: &tcCopy})
		}
		concurrency.Send(ctx, ch, ProviderEvent{Type: EventComplete, Response: resp})
	}()
	return ch
}
