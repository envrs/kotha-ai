package tuicontext

// Permission tracks a pending tool-permission prompt.
type Permission struct {
	Pending bool
	Tool    string
	Detail  string
}

func (p *Permission) Request(tool, detail string) {
	p.Pending = true
	p.Tool = tool
	p.Detail = detail
}

func (p *Permission) Resolve() {
	p.Pending = false
	p.Tool = ""
	p.Detail = ""
}
