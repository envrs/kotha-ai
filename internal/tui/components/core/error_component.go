package core

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// ErrorBox renders a bordered error message.
type ErrorBox struct {
	Title   string
	Message string
	Width   int
}

func (e ErrorBox) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	title := e.Title
	if title == "" {
		title = "Error"
	}
	content := base.Bold(true).Foreground(t.Error()).Render(title) + "\n" +
		base.Foreground(t.Text()).Render(e.Message)
	box := base.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Error()).
		Render(content)
	if e.Width > 0 {
		box = base.Width(e.Width).Render(box)
	}
	return box
}

// ErrorText renders an inline (borderless) error line.
func ErrorText(msg string) string {
	t := theme.CurrentTheme()
	return styles.BaseStyle().Foreground(t.Error()).Render("✖ " + msg)
}
