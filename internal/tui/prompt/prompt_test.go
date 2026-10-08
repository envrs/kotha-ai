package prompt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryNav(t *testing.T) {
	var h History
	h.Add("a")
	h.Add("a") // dup collapses
	h.Add("b")
	if h.Len() != 2 {
		t.Fatalf("len=%d", h.Len())
	}
	if s, ok := h.Prev(); !ok || s != "b" {
		t.Fatalf("prev=%q %v", s, ok)
	}
	if s, ok := h.Prev(); !ok || s != "a" {
		t.Fatalf("prev2=%q %v", s, ok)
	}
	if _, ok := h.Prev(); ok {
		t.Fatal("expected bottom")
	}
	if s, ok := h.Next(); !ok || s != "b" {
		t.Fatalf("next=%q %v", s, ok)
	}
	if _, ok := h.Next(); ok {
		t.Fatal("expected fresh draft")
	}
}

func TestFrecencyDecays(t *testing.T) {
	now := time.Now()
	f := &Frecency{hits: map[string]int{}, last: map[string]time.Time{}, now: func() time.Time { return now }}
	f.Record("a")
	f.Record("a")
	f.Record("b")
	if f.Score("a") <= f.Score("b") {
		t.Fatal("a should outrank b")
	}
	f.now = func() time.Time { return now.Add(100 * time.Hour) }
	if f.Score("a") >= 2.0 {
		t.Fatal("expected decay")
	}
	if f.Score("missing") != 0 {
		t.Fatal("expected zero")
	}
}

func TestIndexMatch(t *testing.T) {
	var ix Index
	ix.Add("src/main.go", "src/util.go", "README.md")
	got := ix.Match("smn", 10)
	if len(got) != 1 || got[0] != "src/main.go" {
		t.Fatalf("match=%v", got)
	}
	if len(ix.Match("", 2)) != 2 {
		t.Fatal("expected limit")
	}
}

func TestMentionAndAutocomplete(t *testing.T) {
	m := New("/tmp")
	m.Workspace.AddFiles("src/main.go", "src/other.go")
	m.Text = "look at @src/ma"
	m.RefreshCandidates(10)
	if !m.Completing || len(m.Candidates) == 0 {
		t.Fatalf("no candidates: %+v", m)
	}
	if !m.AcceptCandidate() || m.Completing {
		t.Fatal("accept failed")
	}
	m.Text = "no token here"
	m.RefreshCandidates(10)
	if m.Completing {
		t.Fatal("should not complete")
	}
}

func TestMoveHelpers(t *testing.T) {
	buf := []rune("hello world\nsecond")
	if PrevWordStart(buf, 11) != 6 {
		t.Fatalf("prev=%d", PrevWordStart(buf, 11))
	}
	if NextWordEnd(buf, 0) != 5 {
		t.Fatalf("next=%d", NextWordEnd(buf, 0))
	}
	if LineStart(buf, 8) != 0 || LineEnd(buf, 0) != 11 {
		t.Fatal("line bounds wrong")
	}
}

func TestStash(t *testing.T) {
	var s Stash
	s.Save("a", Draft{Text: "hi"})
	d, ok := s.Load("a")
	if !ok || d.Text != "hi" {
		t.Fatal("stash broken")
	}
	s.Drop("a")
	if _, ok := s.Load("a"); ok {
		t.Fatal("expected drop")
	}
}

func TestAttachments(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var l AttachmentList
	cwd := NewCwd(dir)
	if err := l.Add(cwd, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := l.Add(cwd, "a.txt"); err != nil || len(l.Items()) != 1 {
		t.Fatal("dedup broken")
	}
	if err := l.Add(cwd, "missing.txt"); err == nil {
		t.Fatal("expected missing error")
	}
	if err := l.Add(cwd, "."); err == nil {
		t.Fatal("expected dir error")
	}
	l.RemoveAt(0)
	if len(l.Items()) != 0 || l.Names() != "" {
		t.Fatal("remove broken")
	}
}

func TestModelSubmitAndSelection(t *testing.T) {
	m := New("/tmp")
	m.Text = "hello"
	if sent := m.Submit(); sent != "hello" || m.History.Len() != 1 || m.Text != "" {
		t.Fatal("submit broken")
	}
	m.Workspace.AddFiles("b", "a")
	m.Text = "@"
	m.RefreshCandidates(10)
	m.MoveSelection(1)
	if m.SelectedIdx != 1 {
		t.Fatal("selection broken")
	}
	m.MoveSelection(1)
	if m.SelectedIdx != 0 {
		t.Fatal("wraparound broken")
	}
}
