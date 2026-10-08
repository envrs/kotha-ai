package core

import (
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// TodoItem is a single checklist row.
type TodoItem struct {
	Title string
	Done  bool
}

func (t TodoItem) Toggle() TodoItem {
	t.Done = !t.Done
	return t
}

func (t TodoItem) View() string {
	th := theme.CurrentTheme()
	base := styles.BaseStyle()
	box := "○"
	style := base.Foreground(th.TextMuted())
	if t.Done {
		box = "●"
		style = base.Foreground(th.Success())
	}
	return style.Render(box + " " + t.Title)
}

// TodoProgress returns "done/total".
func TodoProgress(items []TodoItem) (done, total int) {
	total = len(items)
	for _, it := range items {
		if it.Done {
			done++
		}
	}
	return done, total
}
