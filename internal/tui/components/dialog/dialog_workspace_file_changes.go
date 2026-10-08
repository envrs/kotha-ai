package dialog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type FileChange struct {
	Path   string
	Status rune // A=added, M=modified, D=deleted, R=renamed, ?=untracked
}

type CloseWorkspaceFileChangesDialogMsg struct{}

type WorkspaceFileChangesDialog interface {
	tea.Model
	layout.Bindings
	SetChanges(changes []FileChange)
}

type workspaceFileChangesCmp struct {
	changes []FileChange
	idx     int
}

var workspaceFileChangesKeys = struct {
	Up, Down, Close key.Binding
}{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous file")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next file")),
	Close: key.NewBinding(key.WithKeys("esc", "enter"), key.WithHelp("esc", "close")),
}

func (c *workspaceFileChangesCmp) Init() tea.Cmd { return nil }

func (c *workspaceFileChangesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, workspaceFileChangesKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, workspaceFileChangesKeys.Down):
			if c.idx < len(c.changes)-1 {
				c.idx++
			}
		case key.Matches(msg, workspaceFileChangesKeys.Close):
			return c, func() tea.Msg { return CloseWorkspaceFileChangesDialogMsg{} }
		}
	}
	return c, nil
}

func (c *workspaceFileChangesCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Bold(true).Foreground(t.Primary()).Render("Workspace changes") + "\n"
	for i, ch := range c.changes {
		line := string(ch.Status) + " " + ch.Path
		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+line) + "\n"
		} else {
			out += "  " + line + "\n"
		}
	}
	if len(c.changes) == 0 {
		out += base.Foreground(t.TextMuted()).Render("  working tree clean") + "\n"
	}
	return out
}

func (c *workspaceFileChangesCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(workspaceFileChangesKeys)
}

func (c *workspaceFileChangesCmp) SetChanges(changes []FileChange) {
	c.changes = changes
	c.idx = 0
}

func NewWorkspaceFileChangesCmp() WorkspaceFileChangesDialog {
	return &workspaceFileChangesCmp{}
}
