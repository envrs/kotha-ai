package tuicontext

// Sync tracks dirty/busy state for background refresh.
type Sync struct {
	Dirty     bool
	Syncing   bool
	Version   int64
	LastError string
}

func (s *Sync) MarkDirty() { s.Dirty = true }

func (s *Sync) Begin() {
	s.Syncing = true
	s.Dirty = false
}

func (s *Sync) Done(err error) {
	s.Syncing = false
	s.Version++
	if err != nil {
		s.LastError = err.Error()
	} else {
		s.LastError = ""
	}
}

func (s Sync) NeedsSync() bool { return s.Dirty && !s.Syncing }
