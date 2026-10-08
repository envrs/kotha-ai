package plugin

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/components/dialog"
	"github.com/kothagpt/kotha/internal/tui/util"
)

// CommandShim executes plugin commands through the standard dialog
// command path, so palette, keybindings and the command dialog share one
// execution semantic (including error reporting).
type CommandShim struct {
	Runtime *Runtime
}

func (s CommandShim) Run(id string) tea.Cmd {
	if s.Runtime == nil {
		return util.ReportWarn("plugin runtime not attached")
	}
	return s.Runtime.RunCommand(id)
}

// AsDialogHandler returns a dialog.Command handler that runs the plugin
// command with the given ID.
func (s CommandShim) AsDialogHandler(id string) func(dialog.Command) tea.Cmd {
	return func(_ dialog.Command) tea.Cmd {
		return s.Run(id)
	}
}

// ShimCommand builds a dialog.Command backed by a plugin command ID.
func (s CommandShim) ShimCommand(cmd Command) dialog.Command {
	d := ToDialogCommand(cmd)
	id := cmd.ID
	d.Handler = s.AsDialogHandler(id)
	return d
}
