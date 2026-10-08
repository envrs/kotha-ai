package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type RetryConfirmedMsg struct{ Retry bool }

type RetryActionDialog interface {
	tea.Model
	layout.Bindings
	SetAction(action string)
}

type retryActionCmp struct {
	action   string
	retryYes bool
}

var retryActionKeys = struct {
	Toggle, Confirm, Close key.Binding
}{
	Toggle:  key.NewBinding(key.WithKeys("left", "right", "tab"), key.WithHelp("←/→", "switch")),
	Confirm: key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "confirm")),
	Close:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *retryActionCmp) Init() tea.Cmd { return nil }

func (c *retryActionCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, retryActionKeys.Toggle):
			c.retryYes = !c.retryYes
		case key.Matches(msg, retryActionKeys.Confirm):
			v := c.retryYes
			return c, func() tea.Msg { return RetryConfirmedMsg{Retry: v} }
		case key.Matches(msg, retryActionKeys.Close):
			return c, func() tea.Msg { return RetryConfirmedMsg{Retry: false} }
		}
	}
	return c, nil
}

func (c *retryActionCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	retry := base.Render("[ Retry ]")
	cancel := base.Render("[ Cancel ]")
	if c.retryYes {
		retry = base.Background(t.Primary()).Foreground(t.Background()).Render("[ Retry ]")
	} else {
		cancel = base.Background(t.Primary()).Foreground(t.Background()).Render("[ Cancel ]")
	}
	action := c.action
	if action == "" {
		action = "The action failed."
	}
	return base.Render(action) + "\n" + retry + " " + cancel
}

func (c *retryActionCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(retryActionKeys)
}

func (c *retryActionCmp) SetAction(action string) { c.action = action }

func NewRetryActionCmp() RetryActionDialog {
	return &retryActionCmp{retryYes: true}
}
