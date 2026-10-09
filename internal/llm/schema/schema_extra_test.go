package schema

import (
	"errors"
	"testing"
	"time"
)

func TestRoleValid(t *testing.T) {
	if !RoleSystem.Valid() {
		t.Fatal("RoleSystem should be valid")
	}
	if Role("nope").Valid() {
		t.Fatal("unknown role should be invalid")
	}
}

func TestContentBlockHelpers(t *testing.T) {
	if TextBlock("hi").Type != BlockText {
		t.Fatal("TextBlock type wrong")
	}
	if ReasoningBlock("r").Type != BlockReasoning {
		t.Fatal("ReasoningBlock type wrong")
	}
	if ImageBlock("u").Type != BlockImageURL {
		t.Fatal("ImageBlock type wrong")
	}
}

func TestMessageAccessors(t *testing.T) {
	m := Message{
		Role: RoleAssistant,
		Blocks: []ContentBlock{
			TextBlock("hello"),
			{Type: BlockToolCall, ID: "c1", Name: "tool", Input: "{}"},
			{Type: BlockToolResult, ToolCallID: "c1", Content: "out"},
		},
	}
	if m.Text() != "hello" {
		t.Fatalf("Text() = %q", m.Text())
	}
	if len(m.ToolCalls()) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(m.ToolCalls()))
	}
	if len(m.ToolResults()) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(m.ToolResults()))
	}
}

func TestMessageValidateFailures(t *testing.T) {
	if (Message{Role: Role("bad")}).Validate() == nil {
		t.Fatal("expected invalid role error")
	}
	if (Message{Role: RoleUser, Blocks: []ContentBlock{{Type: BlockToolResult, ToolCallID: ""}}}).Validate() == nil {
		t.Fatal("expected tool result without id error")
	}
	if (Message{Role: RoleUser, Blocks: []ContentBlock{{Type: "weird"}}}).Validate() == nil {
		t.Fatal("expected unknown block type error")
	}
}

func TestStreamEventHelpers(t *testing.T) {
	if ContentDelta("x").Type != StreamContentDelta {
		t.Fatal("ContentDelta type wrong")
	}
	c := Completion{Text: "done"}
	ce := CompleteEvent(c)
	if ce.Type != StreamComplete || ce.Event == nil {
		t.Fatal("CompleteEvent wrong")
	}
	if !ce.IsTerminal() {
		t.Fatal("complete event should be terminal")
	}
	if ErrorEvent(errors.New("boom")).IsTerminal() == false {
		t.Fatal("error event should be terminal")
	}
	if ContentDelta("x").IsTerminal() {
		t.Fatal("delta event should not be terminal")
	}
}

func TestIDsExtra(t *testing.T) {
	for _, id := range []string{"", "  "} {
		if _, err := ParseSessionID(id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID for %q", id)
		}
		if _, err := ParseMessageID(id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID for %q", id)
		}
		if _, err := ParseToolCallID(id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID for %q", id)
		}
		if _, err := ParseRequestID(id); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID for %q", id)
		}
	}
	id, err := ParseSessionID("abc")
	if err != nil || id.Validate() != nil {
		t.Fatal("valid session id should validate")
	}
	if NewToolCallID().String() == "" {
		t.Fatal("NewToolCallID should not be empty")
	}
}

func TestLLMError(t *testing.T) {
	le := NewError(CodeRateLimited, ErrRateLimited).WithRetryAfter(time.Second)
	if !le.Retryable {
		t.Fatal("WithRetryAfter should set retryable")
	}
	if le.RetryAfter != time.Second {
		t.Fatal("RetryAfter wrong")
	}
	if !errors.Is(le, ErrRateLimited) {
		t.Fatal("LLMError should wrap ErrRateLimited")
	}
	if !IsRetryable(le) {
		t.Fatal("expected retryable")
	}
	if CodeOf(le) != CodeRateLimited {
		t.Fatalf("CodeOf = %s", CodeOf(le))
	}
	if !errors.Is(le.Unwrap(), ErrRateLimited) {
		t.Fatal("Unwrap should return ErrRateLimited")
	}
	if CodeOf(ErrAuth) != CodeAuth {
		t.Fatal("CodeOf(ErrAuth) wrong")
	}
	if CodeOf(nil) != CodeUnknown {
		t.Fatal("CodeOf(nil) should be CodeUnknown")
	}
}

func TestWrapHelpers(t *testing.T) {
	wrap := WrapProvider("p", "m", ErrAuth)
	if CodeOf(wrap) != CodeProvider {
		t.Fatalf("expected CodeProvider, got %s", CodeOf(wrap))
	}
	if IsRetryable(wrap) {
		t.Fatal("WrapProvider should not be retryable")
	}
	rl := WrapRateLimited("p", "m", 0, ErrRateLimited)
	if !IsRetryable(rl) {
		t.Fatal("WrapRateLimited should be retryable")
	}
	if CodeOf(rl) != CodeRateLimited {
		t.Fatalf("expected CodeRateLimited, got %s", CodeOf(rl))
	}
}

func TestRequestOptionsValidate(t *testing.T) {
	if (RequestOptions{}).Validate() == nil {
		t.Fatal("expected model required error")
	}
	if (RequestOptions{Model: "m", MaxTokens: -1}).Validate() == nil {
		t.Fatal("expected negative max tokens error")
	}
	if (RequestOptions{Model: "m", Temperature: floatPtr(-0.5)}).Validate() == nil {
		t.Fatal("expected invalid temperature error")
	}
	if (RequestOptions{Model: "m", MaxRetries: -1}).Validate() == nil {
		t.Fatal("expected negative retries error")
	}
	if (RequestOptions{Model: "m", Tools: []ToolDef{{Name: ""}}}).Validate() == nil {
		t.Fatal("expected empty tool name error")
	}
	if (RequestOptions{Model: "m", Temperature: floatPtr(2.5)}).Validate() == nil {
		t.Fatal("expected temperature too high error")
	}
	if (RequestOptions{Model: "m"}).Validate() != nil {
		t.Fatal("expected valid request options")
	}
}

func floatPtr(f float64) *float64 { return &f }
