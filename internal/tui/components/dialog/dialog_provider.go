package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type ProviderSelectedMsg struct{ Provider string }

type CloseProviderDialogMsg struct{}

type ProviderDialog interface {
	tea.Model
	layout.Bindings
	SetProviders(providers []string)
}

type providerCmp struct {
	providers []string
	idx       int
}

var providerKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous provider")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next provider")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select provider")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *providerCmp) Init() tea.Cmd { return nil }

func (c *providerCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, providerKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, providerKeys.Down):
			if c.idx < len(c.providers)-1 {
				c.idx++
			}
		case key.Matches(msg, providerKeys.Select):
			if len(c.providers) > 0 {
				p := c.providers[c.idx]
				return c, func() tea.Msg { return ProviderSelectedMsg{Provider: p} }
			}
		case key.Matches(msg, providerKeys.Close):
			return c, func() tea.Msg { return CloseProviderDialogMsg{} }
		}
	}
	return c, nil
}

func (c *providerCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Select provider") + "\n"
	for i, p := range c.providers {
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+p) + "\n"
		} else {
			out += "  " + p + "\n"
		}
	}
	return out
}

func (c *providerCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(providerKeys)
}

func (c *providerCmp) SetProviders(providers []string) {
	c.providers = providers
	c.idx = 0
}

func NewProviderCmp() ProviderDialog {
	return &providerCmp{}
}
