package sdk

import (
	"context"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/pubsub"
)

func TestExportNilServiceErrors(t *testing.T) {
	c := NewClient(nil, nil, nil, nil)
	if _, err := c.Export(context.Background(), "s1", "json"); !errors.Is(err, ErrNoSessions) {
		t.Fatalf("want ErrNoSessions, got %v", err)
	}
	c = NewClient(&fakeSession{}, nil, nil, nil)
	if _, err := c.Export(context.Background(), "s1", "json"); !errors.Is(err, ErrNoMessages) {
		t.Fatalf("want ErrNoMessages, got %v", err)
	}
}

func TestExportFormats(t *testing.T) {
	svc := &fakeSession{}
	msgs := &fakeMessageSvc{}
	c := NewClient(svc, msgs, nil, nil)
	out, err := c.Export(context.Background(), "s1", "json")
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("expected json bytes")
	}
	out, err = c.Export(context.Background(), "s1", "text")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("expected text")
	}
	out, err = c.Export(context.Background(), "s1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("default should be json")
	}
	if _, err := c.Export(context.Background(), "s1", "xml"); !errors.Is(err, ErrInvalidExportFormat) {
		t.Fatalf("want invalid format, got %v", err)
	}
}

type fakeMessageSvc struct{}

func (f *fakeMessageSvc) List(_ context.Context, sessionID string) ([]message.Message, error) {
	return []message.Message{{ID: "m1", SessionID: sessionID, Parts: []message.ContentPart{message.TextContent{Text: "hi"}}}}, nil
}
func (f *fakeMessageSvc) Create(_ context.Context, _ string, _ message.CreateMessageParams) (message.Message, error) {
	return message.Message{}, nil
}
func (f *fakeMessageSvc) Update(_ context.Context, _ message.Message) error { return nil }
func (f *fakeMessageSvc) Get(_ context.Context, _ string) (message.Message, error) {
	return message.Message{}, nil
}
func (f *fakeMessageSvc) Delete(_ context.Context, _ string) error                { return nil }
func (f *fakeMessageSvc) DeleteSessionMessages(_ context.Context, _ string) error { return nil }
func (f *fakeMessageSvc) Subscribe(_ context.Context) <-chan pubsub.Event[message.Message] {
	ch := make(chan pubsub.Event[message.Message])
	close(ch)
	return ch
}
