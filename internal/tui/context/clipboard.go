package tuicontext

// Clipboard is a best-effort in-memory clipboard with last-copied tracking.
// Platform clipboard wiring can swap the backend via CopyFunc.
type Clipboard struct {
	LastCopied string
	Copies     int
	CopyFunc   func(string) error
}

func (c *Clipboard) Copy(text string) error {
	if c.CopyFunc != nil {
		if err := c.CopyFunc(text); err != nil {
			return err
		}
	}
	c.LastCopied = text
	c.Copies++
	return nil
}
