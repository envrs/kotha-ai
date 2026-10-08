package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type StashEntry struct {
	Name string
	Text string
}

type StashPickedMsg struct{ Name string }

type StashDroppedMsg struct{ Name string }

type CloseStashDialogMsg struct{}

type StashDialog interface {
	tea.Model
	layout.Bindings
	SetEntries(entries []StashEntry)
}

type stashCmp struct {
	entries []StashEntry
	idx     int
}

var stashKeys = struct {
	Up, Down, Select, Drop, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "restore stash")),
	Drop:   key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d", "drop stash")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *stashCmp) Init() tea.Cmd { return nil }

func (c *stashCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, stashKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, stashKeys.Down):
			if c.idx < len(c.entries)-1 {
				c.idx++
			}
		case key.Matches(msg, stashKeys.Select):
			if len(c.entries) > 0 {
				name := c.entries[c.idx].Name
				return c, func() tea.Msg { return StashPickedMsg{Name: name} }
			}
		case key.Matches(msg, stashKeys.Drop):
			if len(c.entries) > 0 {
				name := c.entries[c.idx].Name
				c.entries = append(c.entries[:c.idx], c.entries[c.idx+1:]...)
				if c.idx >= len(c.entries) && c.idx > 0 {
					c.idx--
				}
				return c, func() tea.Msg { return StashDroppedMsg{Name: name} }
			}
		case key.Matches(msg, stashKeys.Close):
			return c, func() tea.Msg { return CloseStashDialogMsg{} }
		}
	}
	return c, nil
}

func (c *stashCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Stashed drafts") + "\n"
	for i, e := range c.entries {
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+e.Name) + "\n"
		} else {
			out += "  " + e.Name + "\n"
		}
	}
	if len(c.entries) == 0 {
		out += base.Foreground(t.TextMuted()).Render("  nothing stashed") + "\n"
	}
	return out
}

func (c *stashCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(stashKeys)
}

func (c *stashCmp) SetEntries(entries []StashEntry) {
	c.entries = entries
	c.idx = 0
}

func NewStashCmp() StashDialog {
	return &stashCmp{}
}
