package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type PaletteItem struct {
	Title  string
	Action func() tea.Msg
}

type ClosePaletteMsg struct{}

type CommandPalette interface {
	tea.Model
	layout.Bindings
	SetItems(items []PaletteItem)
}

type paletteCmp struct {
	items  []PaletteItem
	filter string
	idx    int
}

var paletteKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous")),
	Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *paletteCmp) visible() []PaletteItem {
	if c.filter == "" {
		return c.items
	}
	q := strings.ToLower(c.filter)
	var out []PaletteItem
	for _, it := range c.items {
		if strings.Contains(strings.ToLower(it.Title), q) {
			out = append(out, it)
		}
	}
	return out
}

func (c *paletteCmp) Init() tea.Cmd { return nil }

func (c *paletteCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, paletteKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
			return c, nil
		case key.Matches(msg, paletteKeys.Down):
			if c.idx < len(c.visible())-1 {
				c.idx++
			}
			return c, nil
		case key.Matches(msg, paletteKeys.Select):
			if vis := c.visible(); len(vis) > 0 && vis[c.idx].Action != nil {
				return c, vis[c.idx].Action
			}
			return c, nil
		case key.Matches(msg, paletteKeys.Close):
			return c, func() tea.Msg { return ClosePaletteMsg{} }
		default:
			if msg.Type == tea.KeyRunes {
				c.filter += string(msg.Runes)
				c.idx = 0
			} else if msg.Type == tea.KeyBackspace && len(c.filter) > 0 {
				c.filter = c.filter[:len(c.filter)-1]
				c.idx = 0
			}
		}
	}
	return c, nil
}

func (c *paletteCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Command palette") + "\n"
	out += base.Foreground(t.TextMuted()).Render("> "+c.filter) + "█\n"
	for i, it := range c.visible() {
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+it.Title) + "\n"
		} else {
			out += "  " + it.Title + "\n"
		}
	}
	return out
}

func (c *paletteCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(paletteKeys)
}

func (c *paletteCmp) SetItems(items []PaletteItem) {
	c.items = items
	c.idx = 0
	c.filter = ""
}

func NewCommandPaletteCmp() CommandPalette {
	return &paletteCmp{}
}
