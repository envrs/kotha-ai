package dialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func enterMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func TestAgentSelect(t *testing.T) {
	d := NewAgentSelectCmp()
	d.SetAgents([]string{"coder", "task"})
	m, cmd := d.Update(enterMsg())
	_ = m
	if cmd == nil {
		t.Fatal("expected select cmd")
	}
	if msg := cmd(); msg.(AgentSelectedMsg).Agent != "coder" {
		t.Fatalf("bad agent: %+v", msg)
	}
}

func TestFilterModels(t *testing.T) {
	got := FilterModels([]string{"gpt-4", "claude-3", "gpt-3.5"}, "GPT")
	if len(got) != 2 {
		t.Fatalf("filter=%v", got)
	}
	if len(FilterModels([]string{"a"}, "")) != 1 {
		t.Fatal("empty query should pass through")
	}
}

func TestPaletteFilterAndSelect(t *testing.T) {
	p := NewCommandPaletteCmp()
	var ran bool
	p.SetItems([]PaletteItem{
		{Title: "Init project", Action: func() tea.Msg { ran = true; return nil }},
		{Title: "Quit", Action: func() tea.Msg { return nil }},
	})
	m, _ := p.Update(keyMsg("init"))
	p = m.(CommandPalette)
	if _, cmd := p.Update(enterMsg()); cmd == nil {
		t.Fatal("expected run cmd")
	} else if msg := cmd(); msg != nil {
		_ = msg
	}
	if !ran {
		t.Fatal("action did not run")
	}
}

func TestSkillFilter(t *testing.T) {
	s := NewSkillCmp()
	s.SetSkills([]Skill{{Name: "review", Description: "code review"}, {Name: "test", Description: "run tests"}})
	m, _ := s.Update(keyMsg("rev"))
	s = m.(SkillDialog)
	view := s.View()
	if !strings.Contains(view, "review") || strings.Contains(view, "run tests") {
		t.Fatalf("filter view wrong:\n%s", view)
	}
}

func TestStashDrop(t *testing.T) {
	s := NewStashCmp()
	s.SetEntries([]StashEntry{{Name: "a", Text: "hi"}})
	drop := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	m, cmd := s.Update(drop)
	_ = m
	if cmd == nil {
		t.Fatal("expected drop cmd")
	}
	if msg := cmd(); msg.(StashDroppedMsg).Name != "a" {
		t.Fatalf("bad drop: %+v", msg)
	}
}

func TestRetryToggle(t *testing.T) {
	r := NewRetryActionCmp()
	r.SetAction("deploy failed")
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m, _ := r.Update(tab)
	r = m.(RetryActionDialog)
	if _, cmd := r.Update(enterMsg()); cmd == nil {
		t.Fatal("expected confirm cmd")
	} else if msg := cmd(); msg.(RetryConfirmedMsg).Retry {
		t.Fatal("expected cancel after toggle")
	}
}

func TestRenameSubmit(t *testing.T) {
	r := NewSessionRenameCmp()
	r.SetSession("s1", "")
	m, _ := r.Update(keyMsg("hi"))
	r = m.(SessionRenameDialog)
	_, cmd := r.Update(enterMsg())
	if cmd == nil {
		t.Fatal("expected submit cmd")
	}
	msg := cmd().(SessionRenameSubmittedMsg)
	if msg.SessionID != "s1" || msg.Title != "hi" {
		t.Fatalf("bad rename: %+v", msg)
	}
}

func TestWorkspaceUnavailable(t *testing.T) {
	w := NewWorkspaceUnavailableCmp()
	w.SetReason("gone")
	if !strings.Contains(w.View(), "gone") {
		t.Fatal("reason missing")
	}
}

func TestThemeListPick(t *testing.T) {
	d := NewThemeListCmp()
	d.SetThemes([]string{"dark", "light"}, "dark")
	_, cmd := d.Update(enterMsg())
	if cmd == nil || cmd().(ThemeListPickedMsg).Theme != "dark" {
		t.Fatal("expected dark pick")
	}
}

func TestMcpToggle(t *testing.T) {
	m := NewMcpCmp()
	m.SetServers([]McpServer{{Name: "fs", Connected: false}})
	_, cmd := m.Update(enterMsg())
	if cmd == nil || cmd().(McpToggledMsg).Name != "fs" {
		t.Fatal("expected toggle")
	}
}
