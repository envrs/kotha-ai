package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type Skill struct {
	Name        string
	Description string
}

type SkillSelectedMsg struct{ Name string }

type CloseSkillDialogMsg struct{}

type SkillDialog interface {
	tea.Model
	layout.Bindings
	SetSkills(skills []Skill)
}

type skillCmp struct {
	skills []Skill
	filter string
	idx    int
}

var skillKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous skill")),
	Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next skill")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select skill")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *skillCmp) visible() []Skill {
	if c.filter == "" {
		return c.skills
	}
	q := strings.ToLower(c.filter)
	var out []Skill
	for _, s := range c.skills {
		if strings.Contains(strings.ToLower(s.Name+" "+s.Description), q) {
			out = append(out, s)
		}
	}
	return out
}

func (c *skillCmp) Init() tea.Cmd { return nil }

func (c *skillCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, skillKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
			return c, nil
		case key.Matches(msg, skillKeys.Down):
			if c.idx < len(c.visible())-1 {
				c.idx++
			}
			return c, nil
		case key.Matches(msg, skillKeys.Select):
			if vis := c.visible(); len(vis) > 0 {
				name := vis[c.idx].Name
				return c, func() tea.Msg { return SkillSelectedMsg{Name: name} }
			}
			return c, nil
		case key.Matches(msg, skillKeys.Close):
			return c, func() tea.Msg { return CloseSkillDialogMsg{} }
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

func (c *skillCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Skills") + "\n"
	out += base.Foreground(t.TextMuted()).Render("filter: "+c.filter) + "\n"
	for i, s := range c.visible() {
		line := s.Name
		if s.Description != "" {
			line += " — " + s.Description
		}
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+line) + "\n"
		} else {
			out += "  " + line + "\n"
		}
	}
	return out
}

func (c *skillCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(skillKeys)
}

func (c *skillCmp) SetSkills(skills []Skill) {
	c.skills = skills
	c.idx = 0
	c.filter = ""
}

func NewSkillCmp() SkillDialog {
	return &skillCmp{}
}
