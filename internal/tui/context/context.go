package tuicontext

// Context aggregates every TUI context module so pages and components
// share one construction site without an import web.
type Context struct {
	Args       Args
	Data       Data
	Directory  Directory
	Location   Location
	Local      Local
	Project    Project
	Permission Permission
	Prompt     Prompt
	Editor     Editor
	Clipboard  Clipboard
	Event      Event
	Sync       Sync
	Runtime    Runtime
	Route      Route
	Thinking   Thinking

	Width  int
	Height int
}

// New builds a Context rooted at cwd.
func New(cwd string) Context {
	dir := NewDirectory(cwd)
	return Context{
		Data:      NewData(),
		Directory: dir,
		Local:     DetectLocal(dir.Cwd),
		Project:   DetectProject(dir.Cwd),
		Editor:    Editor{Mode: EditorModeNormal},
		Route:     Route{Page: "chat"},
	}
}

func (c *Context) SetSize(w, h int) {
	c.Width, c.Height = w, h
}
