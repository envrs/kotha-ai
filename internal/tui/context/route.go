package tuicontext

// Route tracks the current page and overlay stack.
type Route struct {
	Page     string
	Overlays []string
}

func (r *Route) Navigate(page string) {
	r.Page = page
	r.Overlays = nil
}

func (r *Route) PushOverlay(name string) {
	r.Overlays = append(r.Overlays, name)
}

func (r *Route) PopOverlay() string {
	if len(r.Overlays) == 0 {
		return ""
	}
	top := r.Overlays[len(r.Overlays)-1]
	r.Overlays = r.Overlays[:len(r.Overlays)-1]
	return top
}

func (r Route) OverlayOpen() bool { return len(r.Overlays) > 0 }
