package schema

// Canonical roles and finish reasons shared by providers and the agent.

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

type FinishReason string

const (
	FinishEndTurn        FinishReason = "end_turn"
	FinishMaxTokens      FinishReason = "max_tokens"
	FinishToolUse        FinishReason = "tool_use"
	FinishCanceled       FinishReason = "canceled"
	FinishError          FinishReason = "error"
	FinishPermissionDeny FinishReason = "permission_denied"
	FinishUnknown        FinishReason = "unknown"
)

type ContentBlockType string

const (
	BlockText       ContentBlockType = "text"
	BlockReasoning  ContentBlockType = "reasoning"
	BlockImageURL   ContentBlockType = "image_url"
	BlockToolCall   ContentBlockType = "tool_call"
	BlockToolResult ContentBlockType = "tool_result"
)

// ContentBlock is a single typed unit of message content.
type ContentBlock struct {
	Type ContentBlockType `json:"type"`

	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
	ImageURL string `json:"image_url,omitempty"`

	// ToolCall fields.
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input string `json:"input,omitempty"`

	// ToolResult fields.
	ToolCallID string `json:"tool_call_id,omitempty"`
	Content    string `json:"content,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
}

func TextBlock(text string) ContentBlock   { return ContentBlock{Type: BlockText, Text: text} }
func ReasoningBlock(t string) ContentBlock { return ContentBlock{Type: BlockReasoning, Thinking: t} }
func ImageBlock(url string) ContentBlock   { return ContentBlock{Type: BlockImageURL, ImageURL: url} }

// Message is the canonical wire-independent chat message.
type Message struct {
	ID        MessageID      `json:"id,omitempty"`
	SessionID SessionID      `json:"session_id,omitempty"`
	Role      Role           `json:"role"`
	Blocks    []ContentBlock `json:"blocks"`
	Model     string         `json:"model,omitempty"`
}

func NewUserMessage(text string) Message {
	return Message{ID: NewMessageID(), Role: RoleUser, Blocks: []ContentBlock{TextBlock(text)}}
}

func NewAssistantMessage(text string) Message {
	return Message{ID: NewMessageID(), Role: RoleAssistant, Blocks: []ContentBlock{TextBlock(text)}}
}

// Text concatenates all text blocks.
func (m Message) Text() string {
	out := ""
	for _, b := range m.Blocks {
		if b.Type == BlockText {
			out += b.Text
		}
	}
	return out
}

// ToolCalls returns all tool-call blocks.
func (m Message) ToolCalls() []ContentBlock {
	var out []ContentBlock
	for _, b := range m.Blocks {
		if b.Type == BlockToolCall {
			out = append(out, b)
		}
	}
	return out
}

// ToolResults returns all tool-result blocks.
func (m Message) ToolResults() []ContentBlock {
	var out []ContentBlock
	for _, b := range m.Blocks {
		if b.Type == BlockToolResult {
			out = append(out, b)
		}
	}
	return out
}

// Validate checks role validity and that tool results carry IDs.
func (m Message) Validate() error {
	if !m.Role.Valid() {
		return NewError(CodeInvalidRequest, ErrInvalidRequest)
	}
	for _, b := range m.Blocks {
		switch b.Type {
		case BlockText, BlockReasoning, BlockImageURL, BlockToolCall, BlockToolResult:
		default:
			return NewError(CodeInvalidRequest, ErrInvalidRequest)
		}
		if b.Type == BlockToolResult && b.ToolCallID == "" {
			return NewError(CodeInvalidRequest, ErrInvalidRequest)
		}
	}
	return nil
}
