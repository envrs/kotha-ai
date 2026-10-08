package prompt

import "strings"

// History is a navigable submitted-prompt log. Consecutive duplicates
// collapse; navigation never mutates the stored entries.
type History struct {
	entries []string
	cursor  int // len(entries) means "fresh draft"
}

func (h *History) Add(entry string) {
	entry = strings.TrimRight(entry, "\n")
	if entry == "" {
		return
	}
	if n := len(h.entries); n > 0 && h.entries[n-1] == entry {
		h.cursor = len(h.entries)
		return
	}
	h.entries = append(h.entries, entry)
	h.cursor = len(h.entries)
}

func (h *History) Len() int { return len(h.entries) }

// Prev moves toward older entries.
func (h *History) Prev() (string, bool) {
	if len(h.entries) == 0 || h.cursor == 0 {
		return "", false
	}
	h.cursor--
	return h.entries[h.cursor], true
}

// Next moves toward newer entries; reports false at the fresh draft.
func (h *History) Next() (string, bool) {
	if h.cursor >= len(h.entries) {
		return "", false
	}
	h.cursor++
	if h.cursor == len(h.entries) {
		return "", false
	}
	return h.entries[h.cursor], true
}

func (h *History) Reset() { h.cursor = len(h.entries) }
