package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type MoveSessionPickedMsg struct{ SessionID string }

type CloseMoveSessionDialogMsg struct{}

type MoveSessionDialog interface {
	tea.Model
	layout.Bindings
	SetSessions(ids, titles []string)
}

type moveSessionCmp struct {
	ids    []string
	titles []string
	idx    int
}

var moveSessionKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "move here")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *moveSessionCmp) Init() tea.Cmd { return nil }

func (c *moveSessionCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, moveSessionKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, moveSessionKeys.Down):
			if c.idx < len(c.ids)-1 {
				c.idx++
			}
		case key.Matches(msg, moveSessionKeys.Select):
			if len(c.ids) > 0 {
				id := c.ids[c.idx]
				return c, func() tea.Msg { return MoveSessionPickedMsg{SessionID: id} }
			}
		case key.Matches(msg, moveSessionKeys.Close):
			return c, func() tea.Msg { return CloseMoveSessionDialogMsg{} }
		}
	}
	return c, nil
}

func (c *moveSessionCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Move to session") + "\n"
	for i, id := range c.ids {
		title := id
		if i < len(c.titles) && c.titles[i] != "" {
			title = c.titles[i]
		}
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+title) + "\n"
		} else {
			out += "  " + title + "\n"
		}
	}
	return out
}

func (c *moveSessionCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(moveSessionKeys)
}

func (c *moveSessionCmp) SetSessions(ids, titles []string) {
	c.ids = ids
	c.titles = titles
	c.idx = 0
}

func NewMoveSessionCmp() MoveSessionDialog {
	return &moveSessionCmp{}
}
