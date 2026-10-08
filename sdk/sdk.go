// Package sdk is the public Go API for embedding Kotha.
//
// It wraps the session, message, agent and permission services behind a
// small stable surface so external tools do not import internal packages:
//
//	client := sdk.NewClient(sessions, messages, coderAgent, permissions)
//	sess, err := client.CreateSession(ctx, "my task")
//	result, err := client.Ask(ctx, sess.ID, "summarize this repo")
package sdk

import (
	"context"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/kothagpt/kotha/internal/session"
)

// Client is the SDK entry point.
type Client struct {
	Sessions    session.Service
	Messages    message.Service
	Agent       agent.Service
	Permissions permission.Service
}

// NewClient builds a Client over the given services. All services are
// required; a nil service causes later calls using it to fail fast.
func NewClient(
	sessions session.Service,
	messages message.Service,
	coderAgent agent.Service,
	permissions permission.Service,
) *Client {
	return &Client{
		Sessions:    sessions,
		Messages:    messages,
		Agent:       coderAgent,
		Permissions: permissions,
	}
}

// CreateSession creates a titled session.
func (c *Client) CreateSession(ctx context.Context, title string) (session.Session, error) {
	if c.Sessions == nil {
		return session.Session{}, ErrNoSessions
	}
	return c.Sessions.Create(ctx, title)
}

// ListSessions returns all sessions.
func (c *Client) ListSessions(ctx context.Context) ([]session.Session, error) {
	if c.Sessions == nil {
		return nil, ErrNoSessions
	}
	return c.Sessions.List(ctx)
}

// GetSession fetches one session by ID.
func (c *Client) GetSession(ctx context.Context, id string) (session.Session, error) {
	if c.Sessions == nil {
		return session.Session{}, ErrNoSessions
	}
	return c.Sessions.Get(ctx, id)
}

// DeleteSession removes a session.
func (c *Client) DeleteSession(ctx context.Context, id string) error {
	if c.Sessions == nil {
		return ErrNoSessions
	}
	return c.Sessions.Delete(ctx, id)
}

// ListMessages returns a session's messages oldest-first.
func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]message.Message, error) {
	if c.Messages == nil {
		return nil, ErrNoMessages
	}
	return c.Messages.List(ctx, sessionID)
}

// Model reports the coder agent's current model.
func (c *Client) Model() models.Model {
	return c.Agent.Model()
}

// UpdateModel switches the coder agent's model.
func (c *Client) UpdateModel(modelID models.ModelID) (models.Model, error) {
	if c.Agent == nil {
		return models.Model{}, ErrNoAgent
	}
	return c.Agent.Update(config.AgentCoder, modelID)
}

// Cancel aborts in-flight work for a session.
func (c *Client) Cancel(sessionID string) {
	if c.Agent != nil {
		c.Agent.Cancel(sessionID)
	}
}

// AutoApproveSession marks a session trusted so permission prompts pass.
func (c *Client) AutoApproveSession(sessionID string) {
	if c.Permissions != nil {
		c.Permissions.AutoApproveSession(sessionID)
	}
}

// Summarize triggers session summarization.
func (c *Client) Summarize(ctx context.Context, sessionID string) error {
	if c.Agent == nil {
		return ErrNoAgent
	}
	return c.Agent.Summarize(ctx, sessionID)
}
