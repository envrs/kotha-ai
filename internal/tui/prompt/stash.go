package prompt

// Draft is a stashed composer snapshot.
type Draft struct {
	Text        string
	Attachments []string
}

// Stash parks drafts under names (one slot per name, e.g. per session).
type Stash struct {
	drafts map[string]Draft
}

func (s *Stash) Save(name string, d Draft) {
	if s.drafts == nil {
		s.drafts = map[string]Draft{}
	}
	s.drafts[name] = d
}

func (s Stash) Load(name string) (Draft, bool) {
	d, ok := s.drafts[name]
	return d, ok
}

func (s *Stash) Drop(name string) {
	delete(s.drafts, name)
}
