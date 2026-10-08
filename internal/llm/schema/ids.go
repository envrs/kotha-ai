package schema

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Typed IDs used across LLM requests, messages, and tool calls.

type (
	SessionID  string
	MessageID  string
	ToolCallID string
	RequestID  string
)

func NewSessionID() SessionID   { return SessionID(uuid.NewString()) }
func NewMessageID() MessageID   { return MessageID(uuid.NewString()) }
func NewRequestID() RequestID   { return RequestID(uuid.NewString()) }
func NewToolCallID() ToolCallID { return ToolCallID("call_" + uuid.NewString()) }

func (id SessionID) String() string  { return string(id) }
func (id MessageID) String() string  { return string(id) }
func (id ToolCallID) String() string { return string(id) }
func (id RequestID) String() string  { return string(id) }

func (id SessionID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: empty session id", ErrInvalidID)
	}
	return nil
}

func (id MessageID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: empty message id", ErrInvalidID)
	}
	return nil
}

func (id ToolCallID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: empty tool call id", ErrInvalidID)
	}
	return nil
}

func (id RequestID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return fmt.Errorf("%w: empty request id", ErrInvalidID)
	}
	return nil
}

func ParseSessionID(s string) (SessionID, error) {
	id := SessionID(s)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

func ParseMessageID(s string) (MessageID, error) {
	id := MessageID(s)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

func ParseToolCallID(s string) (ToolCallID, error) {
	id := ToolCallID(s)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}

func ParseRequestID(s string) (RequestID, error) {
	id := RequestID(s)
	if err := id.Validate(); err != nil {
		return "", err
	}
	return id, nil
}
