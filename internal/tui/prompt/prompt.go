package prompt

// Model composes every prompt submodule into one construction site.
type Model struct {
	Cwd         Cwd
	Workspace   *Workspace
	History     History
	Frecency    *Frecency
	Complete    Autocomplete
	Attachments AttachmentList
	Stash       Stash
	Text        string
	SelectedIdx int
	Candidates  []Candidate
	Completing  bool
}

func New(cwd string) *Model {
	f := NewFrecency()
	return &Model{
		Cwd:       NewCwd(cwd),
		Workspace: NewWorkspace(cwd),
		Frecency:  f,
		Complete:  Autocomplete{Frecency: f},
	}
}

// RefreshCandidates recomputes @-mention completions for the current text.
func (m *Model) RefreshCandidates(limit int) {
	_, token, ok := MentionToken(m.Text)
	if !ok {
		m.Completing = false
		m.Candidates = nil
		m.SelectedIdx = 0
		return
	}
	m.Completing = true
	m.Candidates = m.Complete.Complete(&m.Workspace.Files, token, limit)
	m.SelectedIdx = 0
}

// AcceptCandidate applies the selected candidate to the draft text.
func (m *Model) AcceptCandidate() bool {
	if !m.Completing || len(m.Candidates) == 0 {
		return false
	}
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Candidates) {
		return false
	}
	m.Text = ApplyCompletion(m.Text, m.Candidates[m.SelectedIdx].Value)
	if m.Frecency != nil {
		m.Frecency.Record(m.Candidates[m.SelectedIdx].Value)
	}
	m.Completing = false
	m.Candidates = nil
	return true
}

// MoveSelection moves the candidate cursor with wraparound.
func (m *Model) MoveSelection(delta int) {
	if len(m.Candidates) == 0 {
		return
	}
	m.SelectedIdx = (m.SelectedIdx + delta + len(m.Candidates)) % len(m.Candidates)
}

// Submit records history and clears the draft, returning the sent text.
func (m *Model) Submit() string {
	sent := m.Text
	m.History.Add(sent)
	m.Text = ""
	m.Completing = false
	m.Candidates = nil
	return sent
}
