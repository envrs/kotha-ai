package plugin

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/tui/components/dialog"
)

// Adapters bridge plugin types to existing TUI component types.

// ToDialogCommand converts a plugin Command into a dialog.Command so the
// command dialog can list and run plugin contributions.
func ToDialogCommand(cmd Command) dialog.Command {
	run := cmd.Run
	return dialog.Command{
		ID:          cmd.ID,
		Title:       cmd.Title,
		Description: cmd.Description,
		Handler: func(_ dialog.Command) tea.Cmd {
			if run == nil {
				return nil
			}
			return run
		},
	}
}

// ToDialogCommands converts all commands for dialog registration.
func ToDialogCommands(cmds []Command) []dialog.Command {
	out := make([]dialog.Command, 0, len(cmds))
	for _, cmd := range cmds {
		out = append(out, ToDialogCommand(cmd))
	}
	return out
}

// ToPaletteItems converts commands into command-palette rows.
func ToPaletteItems(cmds []Command) []dialog.PaletteItem {
	out := make([]dialog.PaletteItem, 0, len(cmds))
	for _, cmd := range cmds {
		cmd := cmd
		out = append(out, dialog.PaletteItem{
			Title: cmd.Title,
			Action: func() tea.Msg {
				if cmd.Run == nil {
					return nil
				}
				return cmd.Run()
			},
		})
	}
	return out
}

// FuncPlugin adapts a name + init func into a Plugin.
type FuncPlugin struct {
	PluginName string
	InitFunc   func(HostAPI) error
}

func (p FuncPlugin) Name() string { return p.PluginName }
func (p FuncPlugin) Init(api HostAPI) error {
	if p.InitFunc == nil {
		return nil
	}
	return p.InitFunc(api)
}
