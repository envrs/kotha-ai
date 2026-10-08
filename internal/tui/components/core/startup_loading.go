package core

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

type StartupLoading struct {
	spinner *Spinner
	message string
}

func NewStartupLoading(message string) *StartupLoading {
	if message == "" {
		message = "Loading…"
	}
	return &StartupLoading{spinner: NewSpinner(message), message: message}
}

func (s *StartupLoading) Init() tea.Cmd {
	return s.spinner.Tick()
}

func (s *StartupLoading) Update(msg tea.Msg) (*StartupLoading, tea.Cmd) {
	switch msg.(type) {
	case SpinnerTickMsg:
		s.spinner.Advance()
		return s, s.spinner.Tick()
	}
	return s, nil
}

func (s *StartupLoading) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	return base.Foreground(t.Text()).Render(s.spinner.View())
}

func (s *StartupLoading) SetMessage(message string) {
	s.message = message
	s.spinner.SetLabel(message)
}
