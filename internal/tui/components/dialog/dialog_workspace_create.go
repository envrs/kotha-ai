package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type WorkspaceCreateSubmittedMsg struct{ Name string }

type CloseWorkspaceCreateDialogMsg struct{}

type WorkspaceCreateDialog interface {
	tea.Model
	layout.Bindings
}

type workspaceCreateCmp struct {
	value string
}

var workspaceCreateKeys = struct {
	Submit, Close key.Binding
}{
	Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "create workspace")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *workspaceCreateCmp) Init() tea.Cmd { return nil }

func (c *workspaceCreateCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, workspaceCreateKeys.Submit):
			v := strings.TrimSpace(c.value)
			if v == "" {
				return c, nil
			}
			return c, func() tea.Msg { return WorkspaceCreateSubmittedMsg{Name: v} }
		case key.Matches(msg, workspaceCreateKeys.Close):
			return c, func() tea.Msg { return CloseWorkspaceCreateDialogMsg{} }
		default:
			if msg.Type == tea.KeyRunes {
				c.value += string(msg.Runes)
			} else if msg.Type == tea.KeyBackspace && len(c.value) > 0 {
				c.value = c.value[:len(c.value)-1]
			}
		}
	}
	return c, nil
}

func (c *workspaceCreateCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	return base.Render(
		base.Bold(true).Foreground(t.Primary()).Render("New workspace") + "\n" +
			c.value + "█",
	)
}

func (c *workspaceCreateCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(workspaceCreateKeys)
}

func NewWorkspaceCreateCmp() WorkspaceCreateDialog {
	return &workspaceCreateCmp{}
}
