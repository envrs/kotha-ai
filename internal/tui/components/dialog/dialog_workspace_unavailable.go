package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type WorkspaceRetryMsg struct{}

type CloseWorkspaceUnavailableDialogMsg struct{}

type WorkspaceUnavailableDialog interface {
	tea.Model
	layout.Bindings
	SetReason(reason string)
}

type workspaceUnavailableCmp struct {
	reason string
}

var workspaceUnavailableKeys = struct {
	Retry, Close key.Binding
}{
	Retry: key.NewBinding(key.WithKeys("r", "enter"), key.WithHelp("r", "retry")),
	Close: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *workspaceUnavailableCmp) Init() tea.Cmd { return nil }

func (c *workspaceUnavailableCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, workspaceUnavailableKeys.Retry):
			return c, func() tea.Msg { return WorkspaceRetryMsg{} }
		case key.Matches(msg, workspaceUnavailableKeys.Close):
			return c, func() tea.Msg { return CloseWorkspaceUnavailableDialogMsg{} }
		}
	}
	return c, nil
}

func (c *workspaceUnavailableCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	title := base.Bold(true).Foreground(t.Warning()).Render("Workspace unavailable")
	reason := c.reason
	if reason == "" {
		reason = "The workspace is not reachable."
	}
	return title + "\n" + base.Foreground(t.Text()).Render(reason)
}

func (c *workspaceUnavailableCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(workspaceUnavailableKeys)
}

func (c *workspaceUnavailableCmp) SetReason(reason string) { c.reason = reason }

func NewWorkspaceUnavailableCmp() WorkspaceUnavailableDialog {
	return &workspaceUnavailableCmp{}
}
