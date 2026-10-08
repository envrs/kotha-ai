package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type ConsoleOrgSubmittedMsg struct{ Org string }

type CloseConsoleOrgDialogMsg struct{}

type ConsoleOrgDialog interface {
	tea.Model
	layout.Bindings
}

type consoleOrgCmp struct {
	value string
}

var consoleOrgKeys = struct {
	Submit, Close key.Binding
}{
	Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save org")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *consoleOrgCmp) Init() tea.Cmd { return nil }

func (c *consoleOrgCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, consoleOrgKeys.Submit):
			v := strings.TrimSpace(c.value)
			if v == "" {
				return c, nil
			}
			return c, func() tea.Msg { return ConsoleOrgSubmittedMsg{Org: v} }
		case key.Matches(msg, consoleOrgKeys.Close):
			return c, func() tea.Msg { return CloseConsoleOrgDialogMsg{} }
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

func (c *consoleOrgCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	return base.Render(
		base.Bold(true).Foreground(t.Primary()).Render("Console org") + "\n" +
			base.Foreground(t.TextMuted()).Render("slug: ") + c.value + "█",
	)
}

func (c *consoleOrgCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(consoleOrgKeys)
}

func NewConsoleOrgCmp() ConsoleOrgDialog {
	return &consoleOrgCmp{}
}
