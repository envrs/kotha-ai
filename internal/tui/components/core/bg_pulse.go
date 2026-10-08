package core

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// BgPulse renders a background pulse phase behind content. Phase advances
// 0..len(phases)-1 and selects a progressively stronger background tint.
type BgPulse struct {
	phase int
	steps int
}

func NewBgPulse(steps int) *BgPulse {
	if steps <= 0 {
		steps = 4
	}
	return &BgPulse{steps: steps}
}

func (b *BgPulse) Advance() {
	b.phase = (b.phase + 1) % b.steps
}

func (b *BgPulse) Phase() int { return b.phase }

func (b *BgPulse) View(content string) string {
	return RenderBgPulse(content, b.phase, b.steps)
}

// RenderBgPulse tints content background by phase. Pure function so pages
// can render pulses without owning component state.
func RenderBgPulse(content string, phase, steps int) string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	if steps <= 0 {
		steps = 4
	}
	p := phase % steps
	bg := t.Background()
	if p == steps-1 {
		bg = t.BackgroundSecondary()
	}
	return base.Background(lipgloss.Color(bg.Light)).Render(content)
}
