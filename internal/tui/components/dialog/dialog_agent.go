package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type AgentSelectedMsg struct{ Agent string }

type CloseAgentDialogMsg struct{}

type AgentSelectDialog interface {
	tea.Model
	layout.Bindings
	SetAgents(agents []string)
}

type agentSelectCmp struct {
	agents []string
	idx    int
	width  int
}

var agentSelectKeys = struct {
	Up, Down, Select, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous agent")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next agent")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select agent")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *agentSelectCmp) Init() tea.Cmd { return nil }

func (c *agentSelectCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, agentSelectKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, agentSelectKeys.Down):
			if c.idx < len(c.agents)-1 {
				c.idx++
			}
		case key.Matches(msg, agentSelectKeys.Select):
			if len(c.agents) > 0 {
				return c, func() tea.Msg { return AgentSelectedMsg{Agent: c.agents[c.idx]} }
			}
		case key.Matches(msg, agentSelectKeys.Close):
			return c, func() tea.Msg { return CloseAgentDialogMsg{} }
		}
	}
	return c, nil
}

func (c *agentSelectCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Select agent") + "\n"
	for i, a := range c.agents {
		line := "  " + a
		if i == c.idx {
			line = base.Background(t.Primary()).Foreground(t.Background()).Render("▸ " + a)
		}
		out += line + "\n"
	}
	return out
}

func (c *agentSelectCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(agentSelectKeys)
}

func (c *agentSelectCmp) SetAgents(agents []string) {
	c.agents = agents
	c.idx = 0
}

func NewAgentSelectCmp() AgentSelectDialog {
	return &agentSelectCmp{}
}
