package tuicontext

import (
	"errors"
	"testing"
	"time"
)

func TestPromptAttachDedup(t *testing.T) {
	var p Prompt
	p.Attach("a.txt")
	p.Attach("a.txt")
	if len(p.Attachments) != 1 || !p.Empty() == false {
		t.Fatalf("dedup failed: %+v", p)
	}
	p.Clear()
	if !p.Empty() {
		t.Fatal("expected empty after clear")
	}
}

func TestLocationParse(t *testing.T) {
	loc := ParseLocation("foo.go:10:3")
	if loc.Path != "foo.go" || loc.Line != 10 || loc.Col != 3 {
		t.Fatalf("bad parse: %+v", loc)
	}
	if loc.String() != "foo.go:10:3" {
		t.Fatalf("bad format: %s", loc.String())
	}
}

func TestRouteOverlayStack(t *testing.T) {
	var r Route
	r.Navigate("chat")
	r.PushOverlay("help")
	r.PushOverlay("quit")
	if !r.OverlayOpen() || r.PopOverlay() != "quit" || r.PopOverlay() != "help" {
		t.Fatal("overlay stack broken")
	}
	if r.OverlayOpen() {
		t.Fatal("expected empty stack")
	}
}

func TestSyncLifecycle(t *testing.T) {
	var s Sync
	s.MarkDirty()
	if !s.NeedsSync() {
		t.Fatal("expected needs sync")
	}
	s.Begin()
	s.Done(errors.New("x"))
	if s.Syncing || s.LastError == "" {
		t.Fatal("expected error recorded")
	}
}

func TestThinkingSpinner(t *testing.T) {
	var th Thinking
	th.Start("working")
	first := th.Next()
	second := th.Next()
	if first == "" || first == second {
		t.Fatalf("expected spinner advance: %q %q", first, second)
	}
	th.Stop()
	if th.Next() != "" {
		t.Fatal("expected empty after stop")
	}
}

func TestHelpers(t *testing.T) {
	if FormatHints([]Hint{{Key: "ctrl+c", Desc: "quit"}}) != "ctrl+c quit" {
		t.Fatal("bad hints")
	}
	if TruncateMiddle("abcdef", 100) != "abcdef" {
		t.Fatal("truncate should be identity")
	}
	var c Clipboard
	if err := c.Copy("x"); err != nil || c.LastCopied != "x" || c.Copies != 1 {
		t.Fatal("clipboard broken")
	}
	var perm Permission
	perm.Request("bash", "rm")
	if !perm.Pending {
		t.Fatal("expected pending")
	}
	perm.Resolve()
	if perm.Pending {
		t.Fatal("expected resolved")
	}
	ev := InfoEvent("hi")
	ev.TTL = time.Nanosecond
	if !ev.Expired(time.Now().Add(time.Second)) {
		t.Fatal("expected expired")
	}
	var d Data = NewData()
	d.Set("k", "v")
	if v, ok := d.Get("k"); !ok || v != "v" {
		t.Fatal("data broken")
	}
	ctx := New("/tmp")
	ctx.SetSize(80, 24)
	if ctx.Width != 80 || ctx.Route.Page != "chat" {
		t.Fatal("context init broken")
	}
	if !(Args{InitialPrompt: "hi"}).HasInitialPrompt() {
		t.Fatal("args broken")
	}
	var rt Runtime
	rt.AgentBusy = true
	if !rt.Blocked() {
		t.Fatal("runtime blocked broken")
	}
	var ed Editor
	ed.Focus()
	ed.Mode = EditorModeInsert
	if !ed.Editing() {
		t.Fatal("editor broken")
	}
}
