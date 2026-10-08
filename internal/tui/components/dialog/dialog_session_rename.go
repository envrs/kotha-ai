package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type SessionRenameSubmittedMsg struct {
	SessionID string
	Title     string
}

type CloseSessionRenameDialogMsg struct{}

type SessionRenameDialog interface {
	tea.Model
	layout.Bindings
	SetSession(id, title string)
}

type sessionRenameCmp struct {
	sessionID string
	value     string
}

var sessionRenameKeys = struct {
	Submit, Close key.Binding
}{
	Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "rename session")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *sessionRenameCmp) Init() tea.Cmd { return nil }

func (c *sessionRenameCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, sessionRenameKeys.Submit):
			v := strings.TrimSpace(c.value)
			if v == "" {
				return c, nil
			}
			return c, func() tea.Msg {
				return SessionRenameSubmittedMsg{SessionID: c.sessionID, Title: v}
			}
		case key.Matches(msg, sessionRenameKeys.Close):
			return c, func() tea.Msg { return CloseSessionRenameDialogMsg{} }
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

func (c *sessionRenameCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	return base.Render(
		base.Bold(true).Foreground(t.Primary()).Render("Rename session") + "\n" +
			c.value + "█",
	)
}

func (c *sessionRenameCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(sessionRenameKeys)
}

func (c *sessionRenameCmp) SetSession(id, title string) {
	c.sessionID = id
	c.value = title
}

func NewSessionRenameCmp() SessionRenameDialog {
	return &sessionRenameCmp{}
}
