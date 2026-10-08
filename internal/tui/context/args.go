package tuicontext

import "strings"

// Args carries launch-time arguments into the TUI.
type Args struct {
	SessionID     string
	InitialPrompt string
	Attachments   []string
}

func (a Args) HasInitialPrompt() bool {
	return strings.TrimSpace(a.InitialPrompt) != ""
}
