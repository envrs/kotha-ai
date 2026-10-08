package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type CloseSessionDeleteFailedDialogMsg struct{}

type SessionDeleteFailedDialog interface {
	tea.Model
	layout.Bindings
	SetError(sessionID, errMsg string)
}

type sessionDeleteFailedCmp struct {
	sessionID string
	errMsg    string
}

var sessionDeleteFailedKeys = struct {
	Close key.Binding
}{
	Close: key.NewBinding(key.WithKeys("enter", "esc", " "), key.WithHelp("enter/esc", "dismiss")),
}

func (c *sessionDeleteFailedCmp) Init() tea.Cmd { return nil }

func (c *sessionDeleteFailedCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, sessionDeleteFailedKeys.Close) {
		return c, func() tea.Msg { return CloseSessionDeleteFailedDialogMsg{} }
	}
	return c, nil
}

func (c *sessionDeleteFailedCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	title := base.Bold(true).Foreground(t.Error()).Render("Could not delete session")
	body := base.Foreground(t.Text()).Render(c.sessionID + ": " + c.errMsg)
	return title + "\n" + body
}

func (c *sessionDeleteFailedCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(sessionDeleteFailedKeys)
}

func (c *sessionDeleteFailedCmp) SetError(sessionID, errMsg string) {
	c.sessionID = sessionID
	c.errMsg = errMsg
}

func NewSessionDeleteFailedCmp() SessionDeleteFailedDialog {
	return &sessionDeleteFailedCmp{}
}
