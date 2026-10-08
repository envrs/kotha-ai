package tuicontext

import (
	"fmt"
	"strings"
)

// Hint is a single key-binding hint for the help bar.
type Hint struct {
	Key  string
	Desc string
}

// FormatHints renders hints as "key desc  key desc".
func FormatHints(hints []Hint) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, fmt.Sprintf("%s %s", h.Key, h.Desc))
	}
	return strings.Join(parts, "  ")
}
