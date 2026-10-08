package dialog

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type CloseStatusDialogMsg struct{}

type StatusDialog interface {
	tea.Model
	layout.Bindings
	SetStatus(entries map[string]string)
}

type statusCmp struct {
	entries map[string]string
}

var statusKeys = struct {
	Close key.Binding
}{
	Close: key.NewBinding(key.WithKeys("esc", "enter", " "), key.WithHelp("esc", "close")),
}

func (c *statusCmp) Init() tea.Cmd { return nil }

func (c *statusCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, statusKeys.Close) {
		return c, func() tea.Msg { return CloseStatusDialogMsg{} }
	}
	return c, nil
}

func (c *statusCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := base.Bold(true).Foreground(t.Primary()).Render("Status") + "\n"
	for _, k := range keys {
		out += fmt.Sprintf("%s: %s\n",
			base.Foreground(t.TextMuted()).Render(k),
			base.Foreground(t.Text()).Render(c.entries[k]))
	}
	return out
}

func (c *statusCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(statusKeys)
}

func (c *statusCmp) SetStatus(entries map[string]string) { c.entries = entries }

func NewStatusDialogCmp() StatusDialog {
	return &statusCmp{entries: map[string]string{}}
}
