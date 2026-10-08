package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type VariantSelectedMsg struct{ Variant string }

type CloseVariantDialogMsg struct{}

type VariantDialog interface {
	tea.Model
	layout.Bindings
	SetVariants(variants []string)
}

type variantCmp struct {
	variants []string
	idx      int
}

var variantKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous variant")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next variant")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select variant")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *variantCmp) Init() tea.Cmd { return nil }

func (c *variantCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, variantKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, variantKeys.Down):
			if c.idx < len(c.variants)-1 {
				c.idx++
			}
		case key.Matches(msg, variantKeys.Select):
			if len(c.variants) > 0 {
				v := c.variants[c.idx]
				return c, func() tea.Msg { return VariantSelectedMsg{Variant: v} }
			}
		case key.Matches(msg, variantKeys.Close):
			return c, func() tea.Msg { return CloseVariantDialogMsg{} }
		}
	}
	return c, nil
}

func (c *variantCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Variants") + "\n"
	for i, v := range c.variants {
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+v) + "\n"
		} else {
			out += "  " + v + "\n"
		}
	}
	return out
}

func (c *variantCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(variantKeys)
}

func (c *variantCmp) SetVariants(variants []string) {
	c.variants = variants
	c.idx = 0
}

func NewVariantCmp() VariantDialog {
	return &variantCmp{}
}
