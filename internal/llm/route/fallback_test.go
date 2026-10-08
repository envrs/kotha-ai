package route

import (
	"context"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// stubBackend fails until succeed is set, recording call counts.
type stubBackend struct {
	chatErr   error
	streamErr error
	calls     int
}

func (s *stubBackend) Chat(_ context.Context, _ []schema.Message, _ ...schema.Option) (schema.Completion, error) {
	s.calls++
	if s.chatErr != nil {
		return schema.Completion{}, s.chatErr
	}
	return schema.Completion{Text: "ok", FinishReason: schema.FinishEndTurn}, nil
}

func (s *stubBackend) Stream(_ context.Context, _ []schema.Message, _ ...schema.Option) (<-chan schema.StreamEvent, error) {
	s.calls++
	if s.streamErr != nil {
		return nil, s.streamErr
	}
	ch := make(chan schema.StreamEvent, 1)
	ch <- schema.CompleteEvent(schema.Completion{Text: "ok"})
	close(ch)
	return ch, nil
}

func TestFallbackChatUsesNextBackend(t *testing.T) {
	a := &stubBackend{chatErr: schema.WrapRateLimited("a", "m", 0, errors.New("429"))}
	b := &stubBackend{}
	res, err := NewFallbackClient(a, b).Chat(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "ok" {
		t.Fatalf("text = %q", res.Text)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Fatalf("calls a=%d b=%d", a.calls, b.calls)
	}
}

func TestFallbackChatStopsOnRequestError(t *testing.T) {
	a := &stubBackend{chatErr: schema.NewError(schema.CodeInvalidRequest, errors.New("bad"))}
	b := &stubBackend{}
	if _, err := NewFallbackClient(a, b).Chat(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if b.calls != 0 {
		t.Fatalf("second backend should not be tried, calls=%d", b.calls)
	}
}

func TestFallbackChatReturnsLastError(t *testing.T) {
	a := &stubBackend{chatErr: schema.WrapProvider("a", "m", errors.New("down"))}
	b := &stubBackend{chatErr: schema.WrapProvider("b", "m", errors.New("down"))}
	_, err := NewFallbackClient(a, b).Chat(context.Background(), nil)
	if err == nil || !errors.Is(err, b.chatErr) {
		t.Fatalf("expected last backend's error chain, got %v", err)
	}
	if b.calls != 1 {
		t.Fatalf("expected both tried, b.calls=%d", b.calls)
	}
}

func TestFallbackChatNoBackends(t *testing.T) {
	if _, err := NewFallbackClient().Chat(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty backend list")
	}
}

func TestFallbackStreamEstablishesOnNextBackend(t *testing.T) {
	a := &stubBackend{streamErr: schema.WrapProvider("a", "m", errors.New("boom"))}
	b := &stubBackend{}
	ch, err := NewFallbackClient(a, b).Stream(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	ev := <-ch
	if ev.Type != schema.StreamComplete {
		t.Fatalf("event = %+v", ev)
	}
}

func TestFallbackable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{schema.WrapRateLimited("p", "m", 0, errors.New("x")), true},
		{schema.WrapProvider("p", "m", errors.New("x")), true},
		{schema.NewError(schema.CodeAuth, errors.New("x")), true},
		{schema.NewError(schema.CodeInvalidRequest, errors.New("x")), false},
		{schema.NewError(schema.CodeContextTooLarge, errors.New("x")), false},
		{schema.NewError(schema.CodeCancelled, errors.New("x")), false},
	}
	for _, c := range cases {
		if got := Fallbackable(c.err); got != c.want {
			t.Fatalf("Fallbackable(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestSelectCheapest(t *testing.T) {
	// gpt-4.1-nano is cheaper than gpt-4.1 and claude-3.7-sonnet.
	got := SelectCheapest([]string{"gpt-4.1", "gpt-4.1-nano", "claude-3.7-sonnet"})
	if got != "gpt-4.1-nano" {
		t.Fatalf("got %q, want gpt-4.1-nano", got)
	}
	if got := SelectCheapest([]string{"nope", "also-nope"}); got != "" {
		t.Fatalf("unknown ids = %q, want empty", got)
	}
}
