package prompt

// Workspace binds a project root to its completion index.
type Workspace struct {
	Root  string
	Files Index
}

func NewWorkspace(root string) *Workspace {
	return &Workspace{Root: root}
}

func (w *Workspace) AddFiles(paths ...string) {
	w.Files.Add(paths...)
}

func (w *Workspace) Complete(query string, limit int) []string {
	return w.Files.Match(query, limit)
}
