package toolcall

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kothagpt/kotha/internal/db"
	"github.com/kothagpt/kotha/internal/pubsub"
)

// ToolCall records a single tool invocation made by the agent.
type ToolCall struct {
	ID          string
	SessionID   string
	MessageID   string
	ToolCallID  string
	ToolName    string
	Input       string
	Output      string
	IsError     bool
	DurationMs  int64
	CreatedAt   int64
}

// Service persists and queries tool invocations.
type Service interface {
	pubsub.Suscriber[ToolCall]
	Create(ctx context.Context, call ToolCall) (ToolCall, error)
	Get(ctx context.Context, id string) (ToolCall, error)
	GetByToolCallID(ctx context.Context, toolCallID string) (ToolCall, error)
	ListBySession(ctx context.Context, sessionID string) ([]ToolCall, error)
	ListByMessage(ctx context.Context, messageID string) ([]ToolCall, error)
	Delete(ctx context.Context, id string) error
	DeleteSessionToolCalls(ctx context.Context, sessionID string) error
}

type service struct {
	*pubsub.Broker[ToolCall]
	q db.Querier
}

func NewService(q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[ToolCall](),
		q:      q,
	}
}

func (s *service) Create(ctx context.Context, call ToolCall) (ToolCall, error) {
	if call.ID == "" {
		call.ID = uuid.New().String()
	}
	if call.CreatedAt == 0 {
		call.CreatedAt = time.Now().UnixMilli()
	}
	var messageID sql.NullString
	if call.MessageID != "" {
		messageID = sql.NullString{String: call.MessageID, Valid: true}
	}
	var input sql.NullString
	if call.Input != "" {
		input = sql.NullString{String: call.Input, Valid: true}
	}
	var output sql.NullString
	if call.Output != "" {
		output = sql.NullString{String: call.Output, Valid: true}
	}
	isError := int64(0)
	if call.IsError {
		isError = 1
	}
	dbCall, err := s.q.CreateToolCall(ctx, db.CreateToolCallParams{
		ID:         call.ID,
		SessionID:  call.SessionID,
		MessageID:  messageID,
		ToolCallID: call.ToolCallID,
		ToolName:   call.ToolName,
		Input:      input,
		Output:     output,
		IsError:    isError,
		DurationMs: call.DurationMs,
	})
	if err != nil {
		return ToolCall{}, err
	}
	created := s.fromDBItem(dbCall)
	s.Publish(pubsub.CreatedEvent, created)
	return created, nil
}

func (s *service) Get(ctx context.Context, id string) (ToolCall, error) {
	dbCall, err := s.q.GetToolCall(ctx, id)
	if err != nil {
		return ToolCall{}, err
	}
	return s.fromDBItem(dbCall), nil
}

func (s *service) GetByToolCallID(ctx context.Context, toolCallID string) (ToolCall, error) {
	dbCall, err := s.q.GetToolCallByToolCallID(ctx, toolCallID)
	if err != nil {
		return ToolCall{}, err
	}
	return s.fromDBItem(dbCall), nil
}

func (s *service) ListBySession(ctx context.Context, sessionID string) ([]ToolCall, error) {
	dbCalls, err := s.q.ListToolCallsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	calls := make([]ToolCall, len(dbCalls))
	for i, dbCall := range dbCalls {
		calls[i] = s.fromDBItem(dbCall)
	}
	return calls, nil
}

func (s *service) ListByMessage(ctx context.Context, messageID string) ([]ToolCall, error) {
	dbCalls, err := s.q.ListToolCallsByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	calls := make([]ToolCall, len(dbCalls))
	for i, dbCall := range dbCalls {
		calls[i] = s.fromDBItem(dbCall)
	}
	return calls, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	call, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteToolCall(ctx, call.ID); err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, call)
	return nil
}

func (s *service) DeleteSessionToolCalls(ctx context.Context, sessionID string) error {
	calls, err := s.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, call := range calls {
		_ = s.q.DeleteToolCall(ctx, call.ID)
	}
	return s.q.DeleteSessionToolCalls(ctx, sessionID)
}

func (s *service) fromDBItem(item db.ToolCall) ToolCall {
	return ToolCall{
		ID:         item.ID,
		SessionID:  item.SessionID,
		MessageID:  item.MessageID.String,
		ToolCallID: item.ToolCallID,
		ToolName:   item.ToolName,
		Input:      item.Input.String,
		Output:     item.Output.String,
		IsError:    item.IsError != 0,
		DurationMs: item.DurationMs,
		CreatedAt:  item.CreatedAt,
	}
}

// MarshalJSON implements json.Marshaler for ToolCall so it can be
// surfaced in API responses without exposing internal sql types.
func (t ToolCall) MarshalJSON() ([]byte, error) {
	type alias ToolCall
	return json.Marshal(alias(t))
}

// UnmarshalJSON implements json.Unmarshaler for ToolCall.
func (t *ToolCall) UnmarshalJSON(data []byte) error {
	type alias ToolCall
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = ToolCall(a)
	return nil
}