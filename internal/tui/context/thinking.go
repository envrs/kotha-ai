package tuicontext

// Thinking tracks the assistant "working" indicator.
type Thinking struct {
	Active  bool
	Message string
	frames  int
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (t *Thinking) Start(msg string) {
	t.Active = true
	t.Message = msg
}

func (t *Thinking) Stop() {
	t.Active = false
	t.Message = ""
	t.frames = 0
}

// Next advances the spinner and returns the current frame view.
func (t *Thinking) Next() string {
	if !t.Active {
		return ""
	}
	frame := spinnerFrames[t.frames%len(spinnerFrames)]
	t.frames++
	if t.Message == "" {
		return frame
	}
	return frame + " " + t.Message
}
