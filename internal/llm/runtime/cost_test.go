package runtime

import (
	"context"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// usageChat returns one completion per call with preset usage records.
type usageChat struct {
	usages []schema.TokenUsage
	calls  int
}

func (u *usageChat) Chat(_ context.Context, _ []schema.Message, _ ...schema.Option) (schema.Completion, error) {
	var usage schema.TokenUsage
	if u.calls < len(u.usages) {
		usage = u.usages[u.calls]
	}
	u.calls++
	c := schema.Completion{Text: "ok", Usage: usage}
	if u.calls < len(u.usages) {
		// More usage records queued: force another turn via a tool call.
		c.ToolCalls = []schema.ToolCall{{ID: "c1", Name: "echo", Input: "{}"}}
		c.FinishReason = schema.FinishToolUse
	} else {
		c.FinishReason = schema.FinishEndTurn
	}
	return c, nil
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("empty = %d, want 0", got)
	}
	if got := EstimateTokens("abcd"); got != 1 {
		t.Fatalf("4 bytes = %d, want 1", got)
	}
	if got := EstimateTokens("abc"); got != 1 {
		t.Fatalf("3 bytes = %d, want 1", got)
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	msgs := []schema.Message{
		schema.NewUserMessage("hello world"),
		schema.NewAssistantMessage(""),
	}
	total := EstimateMessagesTokens(msgs)
	if total <= 0 {
		t.Fatalf("expected positive estimate, got %d", total)
	}
	// Each message carries at least the framing overhead.
	if total < 2*messageOverhead {
		t.Fatalf("estimate %d below overhead floor", total)
	}
}

func TestCostForUnknownModelIsZero(t *testing.T) {
	if got := CostFor("definitely-not-a-model", schema.TokenUsage{Input: 1000, Output: 100}); got != 0 {
		t.Fatalf("unknown model cost = %v, want 0", got)
	}
}

func TestCostForOpenAIUsage(t *testing.T) {
	// gpt-4.1-mini: $0.40/1M in, $0.10/1M cached-in, $1.60/1M out.
	// Cached tokens are a subset of input for OpenAI-style tables.
	u := schema.TokenUsage{Input: 1_000_000, CacheRead: 400_000, Output: 100_000}
	got := CostFor("gpt-4.1-mini", u)
	// (1M - 400k) * 0.40 + 400k * 0.10 + 100k * 1.60, per 1M
	want := 0.6*0.40 + 0.4*0.10 + 0.1*1.60
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("cost = %v, want %v", got, want)
	}
}

func TestCostForAnthropicUsage(t *testing.T) {
	// claude-3.7-sonnet: $3/1M in, $3.75/1M cache-write, $0.30/1M
	// cache-read, $15/1M out. Input excludes cache tokens.
	u := schema.TokenUsage{Input: 1_000_000, CacheCreated: 100_000, CacheRead: 200_000, Output: 50_000}
	got := CostFor("claude-3.7-sonnet", u)
	want := 1*3.0 + 0.1*3.75 + 0.2*0.30 + 0.05*15.0
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("cost = %v, want %v", got, want)
	}
}

func TestRunAccumulatesUsageAndCost(t *testing.T) {
	reg := NewRegistry()
	chat := &usageChat{usages: []schema.TokenUsage{
		{Input: 1_000_000, Output: 100_000},
		{Input: 500_000, Output: 50_000},
	}}
	rt := New(chat, reg, WithModel("gpt-4.1-mini"))
	res, err := rt.Run(context.Background(), []schema.Message{schema.NewUserMessage("hi")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Usage.Input != 1_500_000 || res.Usage.Output != 150_000 {
		t.Fatalf("usage = %+v", res.Usage)
	}
	// gpt-4.1-mini: 1.5 * 0.40 + 0.15 * 1.60 = 0.6 + 0.24
	want := 1.5*0.40 + 0.15*1.60
	if diff := res.Cost - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("cost = %v, want %v", res.Cost, want)
	}
}

func TestRunModelFromRequestOptions(t *testing.T) {
	reg := NewRegistry()
	chat := &usageChat{usages: []schema.TokenUsage{{Input: 1_000_000}}}
	rt := New(chat, reg, WithRequestOpts(schema.WithModel("gpt-4.1-nano")))
	res, err := rt.Run(context.Background(), []schema.Message{schema.NewUserMessage("hi")})
	if err != nil {
		t.Fatal(err)
	}
	// gpt-4.1-nano: 1M input * $0.10/1M = $0.10
	if diff := res.Cost - 0.10; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("cost = %v, want 0.10", res.Cost)
	}
}
