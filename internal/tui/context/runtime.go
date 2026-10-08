package tuicontext

// Runtime tracks agent execution state for overlays and guards.
type Runtime struct {
	AgentBusy      bool
	Compacting     bool
	CompactMessage string
	Model          string
}

func (r Runtime) Blocked() bool { return r.AgentBusy || r.Compacting }
