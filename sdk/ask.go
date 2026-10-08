package sdk

import (
	"context"
	"errors"
	"time"

	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/message"
)

// AskOptions tunes a single Ask call.
type AskOptions struct {
	// Attachments sent alongside the prompt.
	Attachments []message.Attachment
	// Timeout bounds the whole call. Zero means no SDK-side timeout
	// (the agent still enforces its own limits).
	Timeout time.Duration
	// AutoApprove skips permission prompts for the session.
	AutoApprove bool
	// OnEvent observes agent events as they arrive. Returning an error
	// does not stop the run; it is reported alongside the result only
	// via the callback itself.
	OnEvent func(agent.AgentEvent)
}

// AskOption customizes AskOptions.
type AskOption func(*AskOptions)

func WithAttachments(a ...message.Attachment) AskOption {
	return func(o *AskOptions) { o.Attachments = append(o.Attachments, a...) }
}

func WithTimeout(d time.Duration) AskOption {
	return func(o *AskOptions) { o.Timeout = d }
}

func WithAutoApprove() AskOption {
	return func(o *AskOptions) { o.AutoApprove = true }
}

func WithEventCallback(fn func(agent.AgentEvent)) AskOption {
	return func(o *AskOptions) { o.OnEvent = fn }
}

// Result is the settled outcome of Ask.
type Result struct {
	Message message.Message
}

// Ask sends a prompt and blocks until the agent finishes, returning the
// final assistant message. Progress can be observed via WithEventCallback.
func (c *Client) Ask(ctx context.Context, sessionID, prompt string, opts ...AskOption) (Result, error) {
	if c.Agent == nil {
		return Result{}, ErrNoAgent
	}
	var o AskOptions
	for _, fn := range opts {
		fn(&o)
	}
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	if o.AutoApprove {
		c.AutoApproveSession(sessionID)
	}
	events, err := c.Agent.Run(ctx, sessionID, prompt, o.Attachments...)
	if err != nil {
		return Result{}, err
	}
	var last agent.AgentEvent
	for ev := range events {
		last = ev
		if o.OnEvent != nil {
			o.OnEvent(ev)
		}
		if ev.Error != nil {
			return Result{}, ev.Error
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
	}
	if last.Error != nil {
		return Result{}, last.Error
	}
	if last.Message.ID == "" && last.Type == "" {
		return Result{}, errors.New("sdk: agent stream ended without a result")
	}
	return Result{Message: last.Message}, nil
}

// Stream starts a run and returns the raw event channel for custom
// consumers. The caller owns draining the channel.
func (c *Client) Stream(ctx context.Context, sessionID, prompt string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	if c.Agent == nil {
		return nil, ErrNoAgent
	}
	return c.Agent.Run(ctx, sessionID, prompt, attachments...)
}
