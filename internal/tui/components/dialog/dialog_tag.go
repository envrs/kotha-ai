package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type TagSelectedMsg struct{ Tag string }

type CloseTagDialogMsg struct{}

type TagDialog interface {
	tea.Model
	layout.Bindings
	SetTags(tags []string)
}

type tagCmp struct {
	tags   []string
	filter string
	idx    int
}

var tagKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous tag")),
	Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next tag")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select tag")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *tagCmp) visible() []string {
	if c.filter == "" {
		return c.tags
	}
	q := strings.ToLower(c.filter)
	var out []string
	for _, tg := range c.tags {
		if strings.Contains(strings.ToLower(tg), q) {
			out = append(out, tg)
		}
	}
	return out
}

func (c *tagCmp) Init() tea.Cmd { return nil }

func (c *tagCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tagKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
			return c, nil
		case key.Matches(msg, tagKeys.Down):
			if c.idx < len(c.visible())-1 {
				c.idx++
			}
			return c, nil
		case key.Matches(msg, tagKeys.Select):
			if vis := c.visible(); len(vis) > 0 {
				tag := vis[c.idx]
				return c, func() tea.Msg { return TagSelectedMsg{Tag: tag} }
			}
			return c, nil
		case key.Matches(msg, tagKeys.Close):
			return c, func() tea.Msg { return CloseTagDialogMsg{} }
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

func (c *tagCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Tags") + "\n"
	for i, tg := range c.visible() {
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+tg) + "\n"
		} else {
			out += "  " + tg + "\n"
		}
	}
	return out
}

func (c *tagCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(tagKeys)
}

func (c *tagCmp) SetTags(tags []string) {
	c.tags = tags
	c.idx = 0
	c.filter = ""
}

func NewTagCmp() TagDialog {
	return &tagCmp{}
}
