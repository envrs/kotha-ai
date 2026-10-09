package layout

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

type sampleKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
}

func TestKeyMapToSlice(t *testing.T) {
	km := sampleKeyMap{
		Up:    key.NewBinding(key.WithKeys("up")),
		Down:  key.NewBinding(key.WithKeys("down")),
		Left:  key.NewBinding(key.WithKeys("left")),
		Right: key.NewBinding(key.WithKeys("right")),
	}
	bindings := KeyMapToSlice(km)
	if len(bindings) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(bindings))
	}
	for _, b := range bindings {
		if b.Keys() == nil {
			t.Fatal("binding should have keys")
		}
	}
}

func TestKeyMapToSliceNonStruct(t *testing.T) {
	if bindings := KeyMapToSlice("not a struct"); len(bindings) != 0 {
		t.Fatalf("expected 0 bindings, got %d", len(bindings))
	}
}

func TestMax(t *testing.T) {
	if max(1, 2) != 2 {
		t.Fatal("max(1,2) should be 2")
	}
	if max(5, 3) != 5 {
		t.Fatal("max(5,3) should be 5")
	}
	if max(-1, -1) != -1 {
		t.Fatal("max(-1,-1) should be -1")
	}
}