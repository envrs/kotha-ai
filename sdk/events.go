package sdk

import (
	"context"

	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/pubsub"
)

// SubscribeAgent forwards agent events until ctx ends. It is a thin
// helper so SDK users do not import the pubsub package.
func (c *Client) SubscribeAgent(ctx context.Context) <-chan pubsub.Event[agent.AgentEvent] {
	if c.Agent == nil {
		ch := make(chan pubsub.Event[agent.AgentEvent])
		close(ch)
		return ch
	}
	return c.Agent.Subscribe(ctx)
}

// Drain collects a stream to its final event, invoking fn per event.
// It mirrors the Ask loop for consumers that already own a channel.
func Drain(events <-chan agent.AgentEvent, fn func(agent.AgentEvent)) (agent.AgentEvent, error) {
	var last agent.AgentEvent
	for ev := range events {
		last = ev
		if fn != nil {
			fn(ev)
		}
		if ev.Error != nil {
			return last, ev.Error
		}
	}
	return last, nil
}
