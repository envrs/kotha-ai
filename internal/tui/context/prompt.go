package tuicontext

import "strings"

// Prompt holds the in-progress composer draft.
type Prompt struct {
	Text        string
	Attachments []string
}

func (p *Prompt) SetText(s string) { p.Text = s }

func (p *Prompt) Attach(path string) {
	for _, existing := range p.Attachments {
		if existing == path {
			return
		}
	}
	p.Attachments = append(p.Attachments, path)
}

func (p *Prompt) Clear() {
	p.Text = ""
	p.Attachments = nil
}

func (p Prompt) Empty() bool {
	return strings.TrimSpace(p.Text) == "" && len(p.Attachments) == 0
}

// RoughTokens estimates tokens as chars/4 for status display.
func (p Prompt) RoughTokens() int {
	return len([]rune(p.Text)) / 4
}
