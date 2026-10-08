package core

// RenderBgPulseFrame renders one animation frame for an external ticker.
// It is a thin alias-friendly entry point so callers reference a single
// render helper regardless of which component owns the pulse state.
func RenderBgPulseFrame(content string, frame int) string {
	const steps = 4
	return RenderBgPulse(content, frame, steps)
}
