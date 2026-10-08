package core

import (
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

const logoText = `██╗  ██╗ ██████╗ ████████╗██╗  ██╗ █████╗
██║ ██╔╝██╔═══██╗╚══██╔══╝██║  ██║██╔══██╗
█████╔╝ ██║   ██║   ██║   ███████║███████║
██╔═██╗ ██║   ██║   ██║   ██╔══██║██╔══██║
██║  ██╗╚██████╔╝   ██║   ██║  ██║██║  ██║
╚═╝  ╚═╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝`

// Logo renders the ASCII wordmark in the theme primary color.
func Logo() string {
	t := theme.CurrentTheme()
	return styles.BaseStyle().Foreground(t.Primary()).Bold(true).Render(logoText)
}
