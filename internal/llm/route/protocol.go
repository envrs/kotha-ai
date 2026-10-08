package route

import (
	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Protocol envelope types. Transport-agnostic; framing handles the bytes.

// Kind identifies the request operation.
type Kind string

const (
	KindChat   Kind = "chat"
	KindStream Kind = "stream"
)

// ChatRequest is a single routed LLM call.
type ChatRequest struct {
	RequestID schema.RequestID      `json:"request_id"`
	Kind      Kind                  `json:"kind"`
	Endpoint  Endpoint              `json:"endpoint"`
	Messages  []schema.Message      `json:"messages"`
	Options   schema.RequestOptions `json:"options"`
}

// ChatResponse is the non-streaming terminal result.
type ChatResponse struct {
	RequestID  schema.RequestID  `json:"request_id"`
	Completion schema.Completion `json:"completion"`
}

// StreamChunk wraps one streaming event on the wire.
type StreamChunk struct {
	RequestID schema.RequestID   `json:"request_id"`
	Event     schema.StreamEvent `json:"event"`
	Final     bool               `json:"final,omitempty"`
}

// ErrorEnvelope carries a coded failure on the wire.
type ErrorEnvelope struct {
	RequestID schema.RequestID `json:"request_id"`
	Code      schema.ErrorCode `json:"code"`
	Message   string           `json:"message"`
	Retryable bool             `json:"retryable,omitempty"`
}

func (r ChatRequest) Validate() error {
	if err := r.Options.Validate(); err != nil {
		return err
	}
	if len(r.Messages) == 0 {
		return schema.NewError(schema.CodeInvalidRequest, schema.ErrInvalidRequest)
	}
	for _, m := range r.Messages {
		if err := m.Validate(); err != nil {
			return err
		}
	}
	if r.Kind != KindChat && r.Kind != KindStream {
		return schema.NewError(schema.CodeInvalidRequest, schema.ErrInvalidRequest)
	}
	return nil
}
