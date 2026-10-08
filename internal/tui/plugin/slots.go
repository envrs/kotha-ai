package plugin

import (
	"sort"
	"strings"
	"sync"
)

// Slots is a registry of named UI extension points ("statusbar",
// "sidebar", "palette", ...). Items render highest-priority first.
type Slots struct {
	mu    sync.RWMutex
	items map[string][]SlotItem
}

func NewSlots() *Slots {
	return &Slots{items: map[string][]SlotItem{}}
}

// Add contributes an item to a slot. Empty slot names are rejected.
func (s *Slots) Add(slot string, item SlotItem) bool {
	if strings.TrimSpace(slot) == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[slot] = append(s.items[slot], item)
	sort.SliceStable(s.items[slot], func(i, j int) bool {
		return s.items[slot][i].Priority > s.items[slot][j].Priority
	})
	return true
}

// List returns a snapshot of a slot's items in render order.
func (s *Slots) List(slot string) []SlotItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SlotItem(nil), s.items[slot]...)
}

// Render concatenates a slot's views, skipping empty output.
func (s *Slots) Render(slot string, width int) string {
	var parts []string
	for _, item := range s.List(slot) {
		if item.View == nil {
			continue
		}
		if v := item.View(width); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "\n")
}

// Names returns all registered slot names in sorted order.
func (s *Slots) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.items))
	for name := range s.items {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
