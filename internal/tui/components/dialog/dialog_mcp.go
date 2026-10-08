package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type McpServer struct {
	Name      string
	Connected bool
}

type McpToggledMsg struct{ Name string }

type CloseMcpDialogMsg struct{}

type McpDialog interface {
	tea.Model
	layout.Bindings
	SetServers(servers []McpServer)
}

type mcpCmp struct {
	servers []McpServer
	idx     int
}

var mcpKeys = struct {
	Up, Down, Toggle, Close key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous server")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next server")),
	Toggle: key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "toggle server")),
	Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
}

func (c *mcpCmp) Init() tea.Cmd { return nil }

func (c *mcpCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, mcpKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, mcpKeys.Down):
			if c.idx < len(c.servers)-1 {
				c.idx++
			}
		case key.Matches(msg, mcpKeys.Toggle):
			if len(c.servers) > 0 {
				c.servers[c.idx].Connected = !c.servers[c.idx].Connected
				name := c.servers[c.idx].Name
				return c, func() tea.Msg { return McpToggledMsg{Name: name} }
			}
		case key.Matches(msg, mcpKeys.Close):
			return c, func() tea.Msg { return CloseMcpDialogMsg{} }
		}
	}
	return c, nil
}

func (c *mcpCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("MCP servers") + "\n"
	for i, s := range c.servers {
		mark := "○"
		if s.Connected {
			mark = "●"
		}
		line := mark + " " + s.Name
		if i == c.idx {
			line = base.Background(t.Primary()).Foreground(t.Background()).Render("▸ " + line)
		} else {
			line = "  " + line
		}
		out += line + "\n"
	}
	return out
}

func (c *mcpCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(mcpKeys)
}

func (c *mcpCmp) SetServers(servers []McpServer) {
	c.servers = servers
	c.idx = 0
}

func NewMcpCmp() McpDialog {
	return &mcpCmp{}
}
