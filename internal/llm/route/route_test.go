package route

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

func TestFramingRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := ChatRequest{RequestID: "r1", Kind: KindChat, Messages: []schema.Message{schema.NewUserMessage("hi")}}
	if err := Encode(&buf, in); err != nil {
		t.Fatal(err)
	}
	var out ChatRequest
	if err := Decode(&buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.RequestID != in.RequestID || out.Messages[0].Text() != "hi" {
		t.Fatalf("mismatch: %+v", out)
	}
}

type fakeTransport struct {
	resp  ChatResponse
	errs  []error
	calls int
}

func (f *fakeTransport) RoundTrip(_ context.Context, req ChatRequest) (ChatResponse, error) {
	f.calls++
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return ChatResponse{}, err
	}
	f.resp.RequestID = req.RequestID
	return f.resp, nil
}

func (f *fakeTransport) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 2)
	ch <- StreamChunk{RequestID: req.RequestID, Event: schema.ContentDelta("hi")}
	ch <- StreamChunk{RequestID: req.RequestID, Event: schema.CompleteEvent(schema.Completion{Text: "hi"}), Final: true}
	close(ch)
	return ch, nil
}

func TestExecutorRetry(t *testing.T) {
	retryable := schema.WrapRateLimited("p", "m", 0, schema.ErrRateLimited)
	ft := &fakeTransport{
		resp: ChatResponse{Completion: schema.Completion{Text: "ok"}},
		errs: []error{retryable},
	}
	ex := Executor{Transport: ft, Backoff: func(int) time.Duration { return 0 }}
	req := ChatRequest{
		RequestID: schema.NewRequestID(), Kind: KindChat,
		Endpoint: MockEndpoint("m"),
		Messages: []schema.Message{schema.NewUserMessage("hi")},
		Options:  schema.NewRequestOptions(schema.WithModel("m")),
	}
	resp, err := ex.Do(context.Background(), req)
	if err != nil || resp.Completion.Text != "ok" || ft.calls != 2 {
		t.Fatalf("got %v %v calls=%d", resp, err, ft.calls)
	}
}

func TestExecutorNoRetryOnFatal(t *testing.T) {
	ft := &fakeTransport{errs: []error{errors.New("boom")}}
	ex := Executor{Transport: ft, Backoff: func(int) time.Duration { return 0 }}
	req := ChatRequest{
		RequestID: schema.NewRequestID(), Kind: KindChat,
		Endpoint: MockEndpoint("m"),
		Messages: []schema.Message{schema.NewUserMessage("hi")},
		Options:  schema.NewRequestOptions(schema.WithModel("m")),
	}
	if _, err := ex.Do(context.Background(), req); err == nil || ft.calls != 1 {
		t.Fatalf("expected single fatal call, got %v calls=%d", err, ft.calls)
	}
}

func TestClientChatAndStream(t *testing.T) {
	ft := &fakeTransport{resp: ChatResponse{Completion: schema.Completion{Text: "hello"}}}
	c := NewClient(MockEndpoint("m"), ft)
	ctx := context.Background()
	comp, err := c.Chat(ctx, []schema.Message{schema.NewUserMessage("hi")})
	if err != nil || comp.Text != "hello" {
		t.Fatalf("chat: %v %v", comp, err)
	}
	events, err := c.Stream(ctx, []schema.Message{schema.NewUserMessage("hi")})
	if err != nil {
		t.Fatal(err)
	}
	var last schema.StreamEvent
	for ev := range events {
		last = ev
	}
	if last.Type != schema.StreamComplete {
		t.Fatalf("expected complete, got %s", last.Type)
	}
}

func TestAuthApply(t *testing.T) {
	if NoAuth().Scheme != SchemeNone {
		t.Fatal("expected none")
	}
	if FromEnv("DEFINITELY_UNSET_VAR_XYZ", "").Scheme != SchemeNone {
		t.Fatal("expected NoAuth for unset env")
	}
}
