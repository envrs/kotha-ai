package core

import (
	"strings"
	"testing"
	"time"
)

func TestSpinnerAdvance(t *testing.T) {
	s := NewSpinner("loading")
	first := s.View()
	s.Advance()
	if s.View() == first {
		t.Fatal("expected frame change")
	}
}

func TestBgPulse(t *testing.T) {
	p := NewBgPulse(4)
	if p.Phase() != 0 {
		t.Fatal("expected phase 0")
	}
	p.Advance()
	if p.Phase() != 1 {
		t.Fatal("expected phase 1")
	}
	if RenderBgPulseFrame("x", 0) == "" {
		t.Fatal("empty render")
	}
}

func TestRegistry(t *testing.T) {
	r := NewSpinnerRegistry()
	r.Register("a")
	r.Advance("a")
	if r.Frame("a") == "" {
		t.Fatal("empty frame")
	}
	r.Unregister("a")
	if len(r.Names()) != 0 {
		t.Fatal("expected empty")
	}
}

func TestTodoItem(t *testing.T) {
	it := TodoItem{Title: "write tests"}
	if strings.Contains(it.View(), "●") {
		t.Fatal("should be open")
	}
	it = it.Toggle()
	if !strings.Contains(it.View(), "●") {
		t.Fatal("should be done")
	}
	done, total := TodoProgress([]TodoItem{it, {Title: "x"}})
	if done != 1 || total != 2 {
		t.Fatalf("progress=%d/%d", done, total)
	}
}

func TestConnected(t *testing.T) {
	c := NewConnectedTracker(time.Second)
	now := time.Now()
	if c.Connected(now) {
		t.Fatal("should start disconnected")
	}
	c.MarkAlive(now)
	if !c.Connected(now) {
		t.Fatal("should be connected")
	}
	if c.Connected(now.Add(2 * time.Second)) {
		t.Fatal("should be stale")
	}
}

func TestMiscViews(t *testing.T) {
	if Logo() == "" {
		t.Fatal("empty logo")
	}
	if ErrorText("boom") == "" || (ErrorBox{Message: "boom"}).View() == "" {
		t.Fatal("empty error views")
	}
	if (WorkspaceLabel{Name: "w", Root: "/r"}).View() == "" {
		t.Fatal("empty label")
	}
	if (PluginRouteMissing{Route: "/x"}).View() == "" {
		t.Fatal("empty route view")
	}
	l := NewStartupLoading("")
	l.SetMessage("hi")
	if l.View() == "" {
		t.Fatal("empty loading")
	}
}
