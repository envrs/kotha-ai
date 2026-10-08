package core

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type SpinnerTickMsg struct{}

type Spinner struct {
	frame int
	label string
}

func NewSpinner(label string) *Spinner {
	return &Spinner{label: label}
}

func (s *Spinner) Tick() tea.Cmd {
	return func() tea.Msg { return SpinnerTickMsg{} }
}

func (s *Spinner) Advance() {
	s.frame = (s.frame + 1) % len(spinnerFrames)
}

func (s *Spinner) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	out := base.Foreground(t.Primary()).Render(spinnerFrames[s.frame])
	if s.label != "" {
		out += " " + base.Foreground(t.Text()).Render(s.label)
	}
	return out
}

func (s *Spinner) SetLabel(label string) { s.label = label }
