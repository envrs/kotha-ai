package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type WorkspaceEntry struct {
	Name   string
	Root   string
	Active bool
}

type WorkspaceListPickedMsg struct{ Name string }

type CloseWorkspaceListDialogMsg struct{}

type WorkspaceListDialog interface {
	tea.Model
	layout.Bindings
	SetWorkspaces(workspaces []WorkspaceEntry)
}

type workspaceListCmp struct {
	workspaces []WorkspaceEntry
	idx        int
}

var workspaceListKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "switch workspace")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *workspaceListCmp) Init() tea.Cmd { return nil }

func (c *workspaceListCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, workspaceListKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, workspaceListKeys.Down):
			if c.idx < len(c.workspaces)-1 {
				c.idx++
			}
		case key.Matches(msg, workspaceListKeys.Select):
			if len(c.workspaces) > 0 {
				name := c.workspaces[c.idx].Name
				return c, func() tea.Msg { return WorkspaceListPickedMsg{Name: name} }
			}
		case key.Matches(msg, workspaceListKeys.Close):
			return c, func() tea.Msg { return CloseWorkspaceListDialogMsg{} }
		}
	}
	return c, nil
}

func (c *workspaceListCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Workspaces") + "\n"
	for i, w := range c.workspaces {
		line := w.Name
		if w.Active {
			line += " ●"
		}
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+line) + "\n"
		} else {
			out += "  " + line + "\n"
		}
	}
	return out
}

func (c *workspaceListCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(workspaceListKeys)
}

func (c *workspaceListCmp) SetWorkspaces(workspaces []WorkspaceEntry) {
	c.workspaces = workspaces
	c.idx = 0
}

func NewWorkspaceListCmp() WorkspaceListDialog {
	return &workspaceListCmp{}
}
