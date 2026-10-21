// Package sdk is the public Go API for embedding Kotha.
//
// It wraps the session, message, agent and permission services behind a
// small stable surface so external tools do not import internal packages:
//
//	client := sdk.NewClient(sessions, messages, coderAgent, permissions)
//	sess, err := client.CreateSession(ctx, "my task")
//	result, err := client.Ask(ctx, sess.ID, "summarize this repo")
//
// Per-instance tool configuration:
//	// Create a client with only the bash and read tools
//	client := sdk.NewClient(sessions, messages, nil, permissions,
//	    sdk.WithTools(sdk.BashTool, sdk.ReadBashOutputTool))
package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/history"
	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/logging"
	"github.com/kothagpt/kotha/internal/lsp"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/kothagpt/kotha/internal/session"
	"github.com/kothagpt/kotha/internal/toolcall"
)

// Tool is the SDK-facing tool interface. Implementations are provided by the SDK.
type Tool interface {
	// Name returns the tool name.
	Name() string
	// Description returns the tool description.
	Description() string
	// InputSchema returns the JSON schema for the tool input.
	InputSchema() map[string]any
	// Run executes the tool.
	Run(ctx context.Context, input string) (string, error)
}

// internalTool wraps an internal tools.BaseTool for SDK use.
type internalTool struct {
	t tools.BaseTool
}

func (i *internalTool) Name() string        { return i.t.Info().Name }
func (i *internalTool) Description() string { return i.t.Info().Description }
func (i *internalTool) InputSchema() map[string]any {
	return i.t.Info().Parameters
}
func (i *internalTool) Run(ctx context.Context, input string) (string, error) {
	// Parse input as ToolCall
	var call tools.ToolCall
	// The input is expected to be JSON matching the tool's parameter schema
	call.Input = input
	resp, err := i.t.Run(ctx, call)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ToolProvider creates tools for a specific instance.
// Implement this to provide per-instance tool sets.
type ToolProvider interface {
	Tools() []Tool
}

// ClientOption configures a Client.
type ClientOption func(*Client) error

// WithTools sets custom tools for this Client instance.
// If not provided, the default coder agent tools are used.
// The tools are used to create a new agent internally.
func WithTools(tools ...Tool) ClientOption {
	return func(c *Client) error {
		if c.Agent != nil {
			return fmt.Errorf("sdk: cannot use WithTools when Agent is already provided via WithAgent")
		}
		c.customTools = tools
		return nil
	}
}

// WithAgent sets a pre-configured agent for this Client instance.
// Use this when you need full control over agent configuration.
// Mutually exclusive with WithTools.
func WithAgent(a agent.Service) ClientOption {
	return func(c *Client) error {
		if len(c.customTools) > 0 {
			return fmt.Errorf("sdk: cannot use WithAgent when WithTools is already set")
		}
		c.Agent = a
		return nil
	}
}

// Client is the SDK entry point.
type Client struct {
	Sessions     session.Service
	Messages     message.Service
	Agent        agent.Service
	Permissions  permission.Service
	customTools  []Tool
	agentConfig  *agentConfig
	agentMu      sync.Mutex
}

type agentConfig struct {
	configProvider config.ConfigProvider
	logger         logging.Logger
	permissions    permission.Service
	sessions       session.Service
	messages       message.Service
	toolcalls      toolcall.Service
	history        history.Service
	lspClients     map[string]*lsp.Client
}

// NewClient builds a Client over the given services.
// Sessions, Messages, and Permissions are required.
// Agent is optional; if not provided, a default coder agent is created
// using the default tools (or custom tools via WithTools).
// If WithTools is used without a pre-configured agent, agentConfig must be provided.
func NewClient(
	sessions session.Service,
	messages message.Service,
	coderAgent agent.Service,
	permissions permission.Service,
	opts ...ClientOption,
) *Client {
	c := &Client{
		Sessions:    sessions,
		Messages:    messages,
		Agent:       coderAgent,
		Permissions: permissions,
	}
	for _, opt := range opts {
		if opt != nil {
			_ = opt(c) // Ignore error for backward compatibility; validate on first use
		}
	}
	return c
}

// ClientConfig holds optional configuration for agent creation when using WithTools.
type ClientConfig struct {
	ConfigProvider config.ConfigProvider
	Logger         logging.Logger
	ToolCalls      toolcall.Service
	History        history.Service
	LSPClients     map[string]*lsp.Client
}

// WithAgentConfig provides the configuration needed to create an agent
// when using WithTools without a pre-configured agent.
func WithAgentConfig(cfg ClientConfig) ClientOption {
	return func(c *Client) error {
		c.agentConfig = &agentConfig{
			configProvider: cfg.ConfigProvider,
			logger:         cfg.Logger,
			toolcalls:      cfg.ToolCalls,
			history:        cfg.History,
			lspClients:     cfg.LSPClients,
			sessions:       c.Sessions,
			messages:       c.Messages,
			permissions:    c.Permissions,
		}
		return nil
	}
}

// CreateSession creates a titled session.
func (c *Client) CreateSession(ctx context.Context, title string) (session.Session, error) {
	if c == nil || c.Sessions == nil {
		return session.Session{}, ErrNoSessions
	}
	return c.Sessions.Create(ctx, title)
}

// ListSessions returns all sessions.
func (c *Client) ListSessions(ctx context.Context) ([]session.Session, error) {
	if c == nil || c.Sessions == nil {
		return nil, ErrNoSessions
	}
	return c.Sessions.List(ctx)
}

// GetSession fetches one session by ID.
func (c *Client) GetSession(ctx context.Context, id string) (session.Session, error) {
	if c == nil || c.Sessions == nil {
		return session.Session{}, ErrNoSessions
	}
	return c.Sessions.Get(ctx, id)
}

// DeleteSession removes a session.
func (c *Client) DeleteSession(ctx context.Context, id string) error {
	if c == nil || c.Sessions == nil {
		return ErrNoSessions
	}
	return c.Sessions.Delete(ctx, id)
}

// ListMessages returns a session's messages oldest-first.
func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]message.Message, error) {
	if c == nil || c.Messages == nil {
		return nil, ErrNoMessages
	}
	return c.Messages.List(ctx, sessionID)
}

// Export snapshots a session and its messages. format is "json"
// (default) or "text"; the result is the encoded export.
func (c *Client) Export(ctx context.Context, sessionID, format string) ([]byte, error) {
	if c == nil || c.Sessions == nil {
		return nil, ErrNoSessions
	}
	if c.Messages == nil {
		return nil, ErrNoMessages
	}
	sess, err := c.Sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	msgs, err := c.Messages.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	exp := session.NewExport(sess, msgs)
	switch format {
	case "", "json":
		return exp.JSON()
	case "text":
		return []byte(exp.Text()), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidExportFormat, format)
	}
}

// Model reports the coder agent's current model.
func (c *Client) Model() models.Model {
	if c == nil {
		return models.Model{}
	}
	ag, err := c.getAgent()
	if err != nil {
		return models.Model{}
	}
	return ag.Model()
}

// UpdateModel switches the coder agent's model.
func (c *Client) UpdateModel(modelID models.ModelID) (models.Model, error) {
	if c == nil {
		return models.Model{}, ErrNoAgent
	}
	ag, err := c.getAgent()
	if err != nil {
		return models.Model{}, err
	}
	return ag.Update(config.AgentCoder, modelID)
}

// Cancel aborts in-flight work for a session.
func (c *Client) Cancel(sessionID string) {
	if c != nil {
		if ag, err := c.getAgent(); err == nil {
			ag.Cancel(sessionID)
		}
	}
}

// AutoApproveSession marks a session trusted so permission prompts pass.
func (c *Client) AutoApproveSession(sessionID string) {
	if c != nil && c.Permissions != nil {
		c.Permissions.AutoApproveSession(sessionID)
	}
}

// Summarize triggers session summarization.
func (c *Client) Summarize(ctx context.Context, sessionID string) error {
	if c == nil {
		return ErrNoAgent
	}
	ag, err := c.getAgent()
	if err != nil {
		return err
	}
	return ag.Summarize(ctx, sessionID)
}

// getAgent returns the agent, creating it with custom tools if needed.
func (c *Client) getAgent() (agent.Service, error) {
	c.agentMu.Lock()
	defer c.agentMu.Unlock()

	if c.Agent != nil {
		return c.Agent, nil
	}

	// Need to create agent with custom tools
	if c.agentConfig == nil {
		return nil, ErrNoAgent
	}

	// Create agent with custom tools
	internalTools := make([]tools.BaseTool, 0, len(c.customTools))
	for _, t := range c.customTools {
		if it, ok := t.(*internalTool); ok {
			internalTools = append(internalTools, it.t)
		}
	}

	if len(internalTools) == 0 {
		return nil, ErrNoAgent
	}

	// Use the internal agent.NewAgent with the custom tools
	ag, err := agent.NewAgent(
		config.AgentCoder,
		c.agentConfig.sessions,
		c.agentConfig.messages,
		c.agentConfig.toolcalls,
		internalTools,
		c.agentConfig.configProvider,
		c.agentConfig.logger,
	)
	if err != nil {
		return nil, err
	}

	c.Agent = ag
	return ag, nil
}

// Prebuilt tools that embedders can use with WithTools.
// Note: Tools requiring permissions/services will need proper configuration
// when used in production. For testing, nil permissions work for some tools.
var (
	// BashTool executes shell commands.
	BashTool Tool = &internalTool{t: tools.NewBashTool(nil)}
	// ReadBashOutputTool reads live output from background bash commands.
	ReadBashOutputTool Tool = &internalTool{t: tools.NewReadBashOutputTool(nil)}
	// StopBashTool stops background bash commands.
	StopBashTool Tool = &internalTool{t: tools.NewStopBashTool(nil)}
	// EditTool edits files.
	EditTool Tool = &internalTool{t: tools.NewEditTool(nil, nil, nil)}
	// FetchTool fetches URLs.
	FetchTool Tool = &internalTool{t: tools.NewFetchTool(nil)}
	// GlobTool finds files by pattern.
	GlobTool Tool = &internalTool{t: tools.NewGlobTool()}
	// GrepTool searches file contents.
	GrepTool Tool = &internalTool{t: tools.NewGrepTool()}
	// LSTool lists directory contents.
	LSTool Tool = &internalTool{t: tools.NewLsTool()}
	// SourcegraphTool searches code via Sourcegraph.
	SourcegraphTool Tool = &internalTool{t: tools.NewSourcegraphTool()}
	// ViewTool reads file contents.
	ViewTool Tool = &internalTool{t: tools.NewViewTool(nil)}
	// WriteTool writes files.
	WriteTool Tool = &internalTool{t: tools.NewWriteTool(nil, nil, nil)}
	// PatchTool applies patches.
	PatchTool Tool = &internalTool{t: tools.NewPatchTool(nil, nil, nil)}
)
