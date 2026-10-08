package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type SessionEntry struct {
	ID    string
	Title string
}

type SessionListPickedMsg struct{ SessionID string }

type CloseSessionListDialogMsg struct{}

type SessionListDialog interface {
	tea.Model
	layout.Bindings
	SetEntries(entries []SessionEntry)
}

type sessionListCmp struct {
	entries []SessionEntry
	idx     int
}

var sessionListKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open session")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *sessionListCmp) Init() tea.Cmd { return nil }

func (c *sessionListCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, sessionListKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, sessionListKeys.Down):
			if c.idx < len(c.entries)-1 {
				c.idx++
			}
		case key.Matches(msg, sessionListKeys.Select):
			if len(c.entries) > 0 {
				id := c.entries[c.idx].ID
				return c, func() tea.Msg { return SessionListPickedMsg{SessionID: id} }
			}
		case key.Matches(msg, sessionListKeys.Close):
			return c, func() tea.Msg { return CloseSessionListDialogMsg{} }
		}
	}
	return c, nil
}

func (c *sessionListCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Sessions") + "\n"
	for i, e := range c.entries {
		title := e.Title
		if title == "" {
			title = e.ID
		}
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+title) + "\n"
		} else {
			out += "  " + title + "\n"
		}
	}
	if len(c.entries) == 0 {
		out += base.Foreground(t.TextMuted()).Render("  no sessions") + "\n"
	}
	return out
}

func (c *sessionListCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(sessionListKeys)
}

func (c *sessionListCmp) SetEntries(entries []SessionEntry) {
	c.entries = entries
	c.idx = 0
}

func NewSessionListCmp() SessionListDialog {
	return &sessionListCmp{}
}
