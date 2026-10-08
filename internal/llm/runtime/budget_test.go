package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

func sysMsg(text string) schema.Message {
	return schema.Message{ID: schema.NewMessageID(), Role: schema.RoleSystem, Blocks: []schema.ContentBlock{schema.TextBlock(text)}}
}

func bigText(n int) string { return strings.Repeat("x", n) }

func TestTrimToBudgetNoOp(t *testing.T) {
	msgs := []schema.Message{
		schema.NewUserMessage("hi"),
		schema.NewAssistantMessage("hello"),
	}
	out, dropped := TrimToBudget(msgs, 10_000)
	if dropped != 0 || len(out) != len(msgs) {
		t.Fatalf("dropped=%d len=%d", dropped, len(out))
	}
	out, dropped = TrimToBudget(msgs, 0) // no budget configured
	if dropped != 0 || len(out) != 2 {
		t.Fatalf("budget 0 should be a no-op, dropped=%d len=%d", dropped, len(out))
	}
}

func TestTrimToBudgetKeepsSystemAndLast(t *testing.T) {
	msgs := []schema.Message{
		sysMsg("system rules"),
		schema.NewUserMessage(bigText(400)),
		schema.NewAssistantMessage(bigText(400)),
		schema.NewUserMessage(bigText(400)),
	}
	out, dropped := TrimToBudget(msgs, 250)
	if dropped == 0 {
		t.Fatal("expected drops")
	}
	if out[0].Role != schema.RoleSystem {
		t.Fatalf("system must stay first, got %s", out[0].Role)
	}
	if out[len(out)-1].Text() != msgs[len(msgs)-1].Text() {
		t.Fatal("final message must stay")
	}
	if got := EstimateMessagesTokens(out); got > 250 {
		t.Fatalf("estimate %d exceeds budget", got)
	}
}

func TestTrimToBudgetDropsToolChainTogether(t *testing.T) {
	msgs := []schema.Message{
		sysMsg("sys"),
		schema.NewUserMessage("q"),
		{
			ID: schema.NewMessageID(), Role: schema.RoleAssistant,
			Blocks: []schema.ContentBlock{
				schema.TextBlock(bigText(400)),
				{Type: schema.BlockToolCall, ID: "c1", Name: "echo", Input: "{}"},
			},
		},
		{
			ID: schema.NewMessageID(), Role: schema.RoleTool,
			Blocks: []schema.ContentBlock{
				{Type: schema.BlockToolResult, ToolCallID: "c1", Content: bigText(400)},
			},
		},
		schema.NewUserMessage("q2"),
	}
	out, dropped := TrimToBudget(msgs, 120)
	if dropped < 2 {
		t.Fatalf("dropped=%d, want assistant+tool dropped together", dropped)
	}
	for _, m := range out {
		if m.Role == schema.RoleTool {
			t.Fatal("orphan tool result left behind")
		}
		if m.Role == schema.RoleAssistant && len(m.ToolCalls()) > 0 {
			t.Fatal("tool-calling assistant left without its result")
		}
	}
	if out[0].Role != schema.RoleSystem || out[len(out)-1].Text() != "q2" {
		t.Fatal("system or final message lost")
	}
}

func TestTrimToBudgetEssentialOverBudget(t *testing.T) {
	msgs := []schema.Message{
		sysMsg(bigText(4000)), // alone over any small budget
		schema.NewUserMessage("q"),
	}
	out, _ := TrimToBudget(msgs, 100)
	if len(out) != 2 {
		t.Fatalf("essential messages must survive, got %d", len(out))
	}
}

func TestRunWithContextBudget(t *testing.T) {
	chat := &captureChat{}
	rt := New(chat, NewRegistry(), WithContextBudget(60))
	msgs := []schema.Message{
		sysMsg("rules"),
		schema.NewUserMessage(bigText(400)),
		schema.NewUserMessage("final question"),
	}
	if _, err := rt.Run(context.Background(), msgs); err != nil {
		t.Fatal(err)
	}
	if len(chat.got) != 2 {
		t.Fatalf("model saw %d messages, want system+last only", len(chat.got))
	}
	if chat.got[0].Role != schema.RoleSystem || chat.got[1].Text() != "final question" {
		t.Fatalf("unexpected transcript: %+v", chat.got)
	}
}

type captureChat struct{ got []schema.Message }

func (c *captureChat) Chat(_ context.Context, msgs []schema.Message, _ ...schema.Option) (schema.Completion, error) {
	c.got = msgs
	return schema.Completion{Text: "ok", FinishReason: schema.FinishEndTurn}, nil
}
