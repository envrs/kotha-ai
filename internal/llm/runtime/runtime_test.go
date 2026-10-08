package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

type fakeChat struct {
	calls int
}

func (f *fakeChat) Chat(_ context.Context, _ []schema.Message, _ ...schema.Option) (schema.Completion, error) {
	f.calls++
	if f.calls == 1 {
		return schema.Completion{
			Text:         "calling",
			ToolCalls:    []schema.ToolCall{{ID: "c1", Name: "echo", Input: `{"text":"hi"}`}},
			FinishReason: schema.FinishToolUse,
		}, nil
	}
	return schema.Completion{Text: "done", FinishReason: schema.FinishEndTurn}, nil
}

func TestLoopRunsToolAndFinishes(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(ToolFunc{
		Def: schema.ToolDef{Name: "echo", Required: []string{"text"}},
		Fn: func(_ context.Context, raw json.RawMessage) (string, error) {
			var in map[string]string
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", err
			}
			return "echo:" + in["text"], nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rt := New(&fakeChat{}, reg, WithMaxTurns(5))
	res, err := rt.Run(context.Background(), []schema.Message{schema.NewUserMessage("go")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Completion.Text != "done" || res.ToolCalls != 1 || res.Turns != 2 {
		t.Fatalf("unexpected result: %+v", res)
	}
	// Tool result message must be present and successful.
	found := false
	for _, m := range res.Messages {
		for _, b := range m.Blocks {
			if b.Type == schema.BlockToolResult && b.Content == "echo:hi" && !b.IsError {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("missing tool result in %+v", res.Messages)
	}
}

func TestUnknownToolIsErrorBlock(t *testing.T) {
	reg := NewRegistry()
	ex := Executor{Registry: reg}
	results := ex.Execute(context.Background(), []schema.ToolCall{{ID: "x", Name: "nope", Input: "{}"}})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected unknown-tool error, got %+v", results)
	}
	blocks := ToBlocks(results)
	if !blocks[0].IsError {
		t.Fatal("expected error block")
	}
}

func TestValidateInput(t *testing.T) {
	def := schema.ToolDef{Name: "t", Required: []string{"a"}}
	if err := ValidateInput(def, json.RawMessage(`{}`)); !errors.Is(err, schema.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if err := ValidateInput(def, json.RawMessage(`{"a":"1"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateRegister(t *testing.T) {
	reg := NewRegistry()
	tool := ToolFunc{Def: schema.ToolDef{Name: "dup"}}
	if err := reg.Register(tool); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(tool); err == nil {
		t.Fatal("expected duplicate error")
	}
}
