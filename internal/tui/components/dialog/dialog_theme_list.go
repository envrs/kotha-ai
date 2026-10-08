package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type ThemeListPickedMsg struct{ Theme string }

type CloseThemeListDialogMsg struct{}

type ThemeListDialog interface {
	tea.Model
	layout.Bindings
	SetThemes(themes []string, current string)
}

type themeListCmp struct {
	themes  []string
	current string
	idx     int
}

var themeListKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous theme")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next theme")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply theme")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *themeListCmp) Init() tea.Cmd { return nil }

func (c *themeListCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, themeListKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, themeListKeys.Down):
			if c.idx < len(c.themes)-1 {
				c.idx++
			}
		case key.Matches(msg, themeListKeys.Select):
			if len(c.themes) > 0 {
				name := c.themes[c.idx]
				return c, func() tea.Msg { return ThemeListPickedMsg{Theme: name} }
			}
		case key.Matches(msg, themeListKeys.Close):
			return c, func() tea.Msg { return CloseThemeListDialogMsg{} }
		}
	}
	return c, nil
}

func (c *themeListCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Themes") + "\n"
	for i, name := range c.themes {
		marker := "  "
		if name == c.current {
			marker = "● "
		}
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+marker+name) + "\n"
		} else {
			out += "  " + marker + name + "\n"
		}
	}
	return out
}

func (c *themeListCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(themeListKeys)
}

func (c *themeListCmp) SetThemes(themes []string, current string) {
	c.themes = themes
	c.current = current
	c.idx = 0
}

func NewThemeListCmp() ThemeListDialog {
	return &themeListCmp{}
}
