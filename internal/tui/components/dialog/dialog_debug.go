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

type CloseDebugDialogMsg struct{}

type DebugDialog interface {
	tea.Model
	layout.Bindings
	SetLines(lines map[string]string)
}

type debugCmp struct {
	lines   map[string]string
	verbose bool
}

var debugKeys = struct {
	Toggle, Close key.Binding
}{
	Toggle: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle verbose")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *debugCmp) Init() tea.Cmd { return nil }

func (c *debugCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(km, debugKeys.Toggle):
			c.verbose = !c.verbose
		case key.Matches(km, debugKeys.Close):
			return c, func() tea.Msg { return CloseDebugDialogMsg{} }
		}
	}
	return c, nil
}

func (c *debugCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	keys := make([]string, 0, len(c.lines))
	for k := range c.lines {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := base.Bold(true).Foreground(t.Primary()).Render("Debug") + "\n"
	for _, k := range keys {
		out += fmt.Sprintf("%s=%s\n",
			base.Foreground(t.TextMuted()).Render(k),
			base.Foreground(t.Text()).Render(c.lines[k]))
	}
	if c.verbose {
		out += base.Foreground(t.TextMuted()).Render("verbose: on") + "\n"
	}
	return out
}

func (c *debugCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(debugKeys)
}

func (c *debugCmp) SetLines(lines map[string]string) { c.lines = lines }

func NewDebugCmp() DebugDialog {
	return &debugCmp{lines: map[string]string{}}
}
