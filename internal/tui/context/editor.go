package tuicontext

// Editor tracks composer focus and editing mode.
type EditorMode string

const (
	EditorModeNormal EditorMode = "normal"
	EditorModeInsert EditorMode = "insert"
)

type Editor struct {
	Focused bool
	Mode    EditorMode
}

func (e *Editor) Focus()       { e.Focused = true }
func (e *Editor) Blur()        { e.Focused = false }
func (e Editor) Editing() bool { return e.Focused && e.Mode == EditorModeInsert }
