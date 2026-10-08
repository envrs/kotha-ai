package core

import (
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// PluginRouteMissing renders a notice for an unregistered plugin route.
type PluginRouteMissing struct {
	Route string
}

func (p PluginRouteMissing) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	return base.Foreground(t.Warning()).Render("⚠ no plugin handles route: " + p.Route)
}
