package core

import (
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// WorkspaceLabel renders the active workspace name + root.
type WorkspaceLabel struct {
	Name string
	Root string
}

func (w WorkspaceLabel) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	name := w.Name
	if name == "" {
		name = "default"
	}
	out := base.Foreground(t.Primary()).Bold(true).Render("⬢ " + name)
	if w.Root != "" {
		out += " " + base.Foreground(t.TextMuted()).Render("("+w.Root+")")
	}
	return out
}
