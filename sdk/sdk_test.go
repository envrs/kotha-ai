package sdk

import (
	"context"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/kothagpt/kotha/internal/pubsub"
	"github.com/kothagpt/kotha/internal/session"
)

type fakeSession struct {
	sessions []session.Session
}

func (f *fakeSession) Create(_ context.Context, title string) (session.Session, error) {
	return session.Session{ID: "s1", Title: title}, nil
}

func (f *fakeSession) CreateTitleSession(_ context.Context, _ string) (session.Session, error) {
	return session.Session{}, nil
}

func (f *fakeSession) CreateTaskSession(_ context.Context, _, _, _ string) (session.Session, error) {
	return session.Session{}, nil
}

func (f *fakeSession) Get(_ context.Context, id string) (session.Session, error) {
	return session.Session{ID: id}, nil
}

func (f *fakeSession) List(_ context.Context) ([]session.Session, error) {
	return append([]session.Session(nil), f.sessions...), nil
}

func (f *fakeSession) Save(_ context.Context, s session.Session) (session.Session, error) {
	return s, nil
}

func (f *fakeSession) Delete(_ context.Context, _ string) error { return nil }

func (f *fakeSession) Subscribe(_ context.Context) <-chan pubsub.Event[session.Session] {
	ch := make(chan pubsub.Event[session.Session])
	close(ch)
	return ch
}

type fakeAgent struct {
	events chan agent.AgentEvent
}

func (f *fakeAgent) Run(_ context.Context, _, _ string, _ ...message.Attachment) (<-chan agent.AgentEvent, error) {
	return f.events, nil
}

func (f *fakeAgent) Cancel(_ string)             {}
func (f *fakeAgent) IsSessionBusy(_ string) bool { return false }
func (f *fakeAgent) IsBusy() bool                { return false }
func (f *fakeAgent) Model() models.Model         { return models.Model{ID: "mock"} }
func (f *fakeAgent) Update(_ config.AgentName, _ models.ModelID) (models.Model, error) {
	return models.Model{ID: "mock"}, nil
}
func (f *fakeAgent) Summarize(_ context.Context, _ string) error { return nil }
func (f *fakeAgent) Subscribe(_ context.Context) <-chan pubsub.Event[agent.AgentEvent] {
	ch := make(chan pubsub.Event[agent.AgentEvent])
	close(ch)
	return ch
}

type fakePermission struct{ approved bool }

func (f *fakePermission) Grant(_ permission.PermissionRequest)              {}
func (f *fakePermission) GrantPersistant(_ permission.PermissionRequest)    {}
func (f *fakePermission) Deny(_ permission.PermissionRequest)               {}
func (f *fakePermission) Request(_ permission.CreatePermissionRequest) bool { return f.approved }
func (f *fakePermission) AutoApproveSession(_ string)                       {}

func (f *fakePermission) Subscribe(_ context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	ch := make(chan pubsub.Event[permission.PermissionRequest])
	close(ch)
	return ch
}

func TestNilServicesFailFast(t *testing.T) {
	c := NewClient(nil, nil, nil, nil)
	if _, err := c.CreateSession(context.Background(), "x"); !errors.Is(err, ErrNoSessions) {
		t.Fatalf("sessions err: %v", err)
	}
	if _, err := c.ListMessages(context.Background(), "x"); !errors.Is(err, ErrNoMessages) {
		t.Fatalf("messages err: %v", err)
	}
	if _, err := c.UpdateModel("x"); !errors.Is(err, ErrNoAgent) {
		t.Fatalf("agent err: %v", err)
	}
	if err := c.Summarize(context.Background(), "x"); !errors.Is(err, ErrNoAgent) {
		t.Fatalf("summarize err: %v", err)
	}
}

func TestAskDrainsToFinal(t *testing.T) {
	evs := make(chan agent.AgentEvent, 3)
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Progress: "thinking"}
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Message: message.Message{ID: "m1", Parts: []message.ContentPart{message.TextContent{Text: "done"}}}}
	close(evs)
	a := &fakeAgent{events: evs}
	c := NewClient(&fakeSession{}, nil, a, &fakePermission{})
	res, err := c.Ask(context.Background(), "s1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if res.Message.Content().String() != "done" {
		t.Fatalf("result=%+v", res)
	}
}

func TestAskErrorPropagates(t *testing.T) {
	evs := make(chan agent.AgentEvent, 1)
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeError, Error: errors.New("boom")}
	close(evs)
	a := &fakeAgent{events: evs}
	c := NewClient(&fakeSession{}, nil, a, &fakePermission{})
	if _, err := c.Ask(context.Background(), "s1", "hi"); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestAskStream(t *testing.T) {
	evs := make(chan agent.AgentEvent, 2)
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Progress: "thinking"}
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Done: true}
	close(evs)
	a := &fakeAgent{events: evs}
	c := NewClient(&fakeSession{}, nil, a, &fakePermission{})
	stream, err := c.AskStream(context.Background(), "s1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for range stream {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}
}

func TestAskStreamNilAgent(t *testing.T) {
	c := NewClient(&fakeSession{}, nil, nil, &fakePermission{})
	if _, err := c.AskStream(context.Background(), "s1", "hi"); !errors.Is(err, ErrNoAgent) {
		t.Fatalf("want ErrNoAgent, got %v", err)
	}
}

func TestDrain(t *testing.T) {
	evs := make(chan agent.AgentEvent, 2)
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Progress: "p1"}
	evs <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Message: message.Message{ID: "m", Parts: []message.ContentPart{message.TextContent{Text: "done"}}}}
	close(evs)
	var seen []string
	_, err := Drain(evs, func(ev agent.AgentEvent) { seen = append(seen, ev.Progress) })
	if err != nil || len(seen) != 2 || seen[0] != "p1" {
		t.Fatalf("drain=%v %v", seen, err)
	}
}
