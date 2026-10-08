package schema

// Streaming + agent event envelope shared by providers and the agent bus.

type StreamEventType string

const (
	StreamContentStart  StreamEventType = "content_start"
	StreamContentDelta  StreamEventType = "content_delta"
	StreamContentStop   StreamEventType = "content_stop"
	StreamThinkingDelta StreamEventType = "thinking_delta"
	StreamToolUseStart  StreamEventType = "tool_use_start"
	StreamToolUseDelta  StreamEventType = "tool_use_delta"
	StreamToolUseStop   StreamEventType = "tool_use_stop"
	StreamComplete      StreamEventType = "complete"
	StreamError         StreamEventType = "error"
	StreamWarning       StreamEventType = "warning"
)

// TokenUsage mirrors provider accounting in a provider-neutral shape.
type TokenUsage struct {
	Input        int64 `json:"input_tokens"`
	Output       int64 `json:"output_tokens"`
	CacheCreated int64 `json:"cache_creation_tokens"`
	CacheRead    int64 `json:"cache_read_tokens"`
}

// Completion is the terminal result of a turn.
type Completion struct {
	Text         string       `json:"text"`
	ToolCalls    []ToolCall   `json:"tool_calls"`
	Usage        TokenUsage   `json:"usage"`
	FinishReason FinishReason `json:"finish_reason"`
}

// ToolCall is the canonical tool invocation.
type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

// StreamEvent is a single chunk on a streaming response channel.
type StreamEvent struct {
	Type      StreamEventType `json:"type"`
	RequestID RequestID       `json:"request_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ToolCall  *ToolCall       `json:"tool_call,omitempty"`
	Usage     *TokenUsage     `json:"usage,omitempty"`
	Event     *Completion     `json:"completion,omitempty"`
	Err       error           `json:"-"`
	Warning   string          `json:"warning,omitempty"`
}

func ContentDelta(content string) StreamEvent {
	return StreamEvent{Type: StreamContentDelta, Content: content}
}

func CompleteEvent(c Completion) StreamEvent {
	return StreamEvent{Type: StreamComplete, Event: &c, Usage: &c.Usage}
}

func ErrorEvent(err error) StreamEvent {
	return StreamEvent{Type: StreamError, Err: err}
}

func (e StreamEvent) IsTerminal() bool {
	return e.Type == StreamComplete || e.Type == StreamError
}

type AgentEventType string

const (
	AgentResponse  AgentEventType = "response"
	AgentError     AgentEventType = "error"
	AgentSummarize AgentEventType = "summarize"
)

// AgentEvent is published on the agent bus per turn/summary.
type AgentEvent struct {
	Type      AgentEventType `json:"type"`
	SessionID SessionID      `json:"session_id"`
	Message   Message        `json:"message"`
	Progress  string         `json:"progress,omitempty"`
	Done      bool           `json:"done,omitempty"`
	Err       error          `json:"-"`
}
