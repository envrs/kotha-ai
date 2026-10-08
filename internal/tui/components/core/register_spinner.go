package core

import tea "github.com/charmbracelet/bubbletea"

// SpinnerRegistry tracks named spinner ticks so multiple concurrent
// background tasks can share one tick subscription.
type SpinnerRegistry struct {
	frames map[string]int
}

func NewSpinnerRegistry() *SpinnerRegistry {
	return &SpinnerRegistry{frames: map[string]int{}}
}

// Register adds a named spinner and returns its tick command.
func (r *SpinnerRegistry) Register(name string) tea.Cmd {
	if _, ok := r.frames[name]; !ok {
		r.frames[name] = 0
	}
	return func() tea.Msg { return SpinnerTickMsg{} }
}

func (r *SpinnerRegistry) Advance(name string) {
	r.frames[name] = (r.frames[name] + 1) % len(spinnerFrames)
}

func (r *SpinnerRegistry) Frame(name string) string {
	return spinnerFrames[r.frames[name]%len(spinnerFrames)]
}

func (r *SpinnerRegistry) Unregister(name string) {
	delete(r.frames, name)
}

func (r *SpinnerRegistry) Names() []string {
	out := make([]string, 0, len(r.frames))
	for n := range r.frames {
		out = append(out, n)
	}
	return out
}
