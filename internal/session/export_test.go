package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kothagpt/kotha/internal/message"
)

func testExport() Export {
	sess := Session{ID: "s1", Title: "My task"}
	msgs := []message.Message{
		{ID: "m1", Role: message.User, SessionID: "s1", CreatedAt: 100,
			Parts: []message.ContentPart{message.TextContent{Text: "hello"}}},
		{ID: "m2", Role: message.Assistant, SessionID: "s1", CreatedAt: 200,
			Parts: []message.ContentPart{message.TextContent{Text: "hi there"}}},
		{ID: "m3", Role: message.Assistant, SessionID: "s1", CreatedAt: 300,
			Parts: []message.ContentPart{message.ReasoningContent{Thinking: "hidden"}}},
	}
	return NewExport(sess, msgs)
}

func TestNewExport(t *testing.T) {
	e := testExport()
	if e.SessionID != "s1" || e.Title != "My task" {
		t.Fatalf("export = %+v", e)
	}
	if len(e.Messages) != 3 {
		t.Fatalf("messages = %d", len(e.Messages))
	}
	if e.Messages[0].Role != "user" || e.Messages[0].Content != "hello" {
		t.Fatalf("msg0 = %+v", e.Messages[0])
	}
	// Reasoning-only messages export as empty text, not dropped.
	if e.Messages[2].Content != "" || e.Messages[2].Role != "assistant" {
		t.Fatalf("msg2 = %+v", e.Messages[2])
	}
}

func TestExportJSON(t *testing.T) {
	b, err := testExport().JSON()
	if err != nil {
		t.Fatal(err)
	}
	var round Export
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if round.SessionID != "s1" || len(round.Messages) != 3 {
		t.Fatalf("round trip = %+v", round)
	}
	if round.Messages[1].Content != "hi there" {
		t.Fatalf("content lost: %+v", round.Messages[1])
	}
}

func TestExportText(t *testing.T) {
	out := testExport().Text()
	if !strings.HasPrefix(out, "My task\n====") {
		t.Fatalf("missing title header: %q", out)
	}
	for _, want := range []string{"[user]\nhello", "[assistant]\nhi there"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hidden") {
		t.Fatal("reasoning must not leak into text export")
	}
}

func TestExportTextEmptyTitleFallsBackToID(t *testing.T) {
	e := NewExport(Session{ID: "only-id"}, nil)
	if !strings.HasPrefix(e.Text(), "only-id\n") {
		t.Fatalf("expected id header, got %q", e.Text())
	}
}
