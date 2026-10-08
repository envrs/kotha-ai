package plugin

import tea "github.com/charmbracelet/bubbletea"

// Plugin is the interface every TUI plugin implements.
type Plugin interface {
	// Name uniquely identifies the plugin (e.g. "auth", "compact").
	Name() string
	// Init lets the plugin register commands, slots and routes.
	Init(api HostAPI) error
}

// HostAPI is the capability surface the TUI hands to plugins.
type HostAPI interface {
	// RegisterCommand adds a palette/dialog command.
	RegisterCommand(cmd Command)
	// RegisterSlot contributes a view to a named slot.
	RegisterSlot(slot string, item SlotItem)
	// RegisterRoute maps a route to a handler command factory.
	RegisterRoute(route string, handler RouteHandler)
	// Notify surfaces info/warn/error status messages.
	NotifyInfo(msg string) tea.Cmd
	NotifyWarn(msg string) tea.Cmd
	NotifyError(err error) tea.Cmd
}

// Command is a plugin-provided command (mirrors dialog.Command shape
// without importing the dialog package).
type Command struct {
	ID          string
	Title       string
	Description string
	Run         func() tea.Msg
}

// SlotItem is one contribution to a named UI slot.
type SlotItem struct {
	Plugin   string
	Priority int
	View     func(width int) string
}

// RouteHandler builds the command that serves a plugin route.
type RouteHandler func(route string) tea.Cmd
