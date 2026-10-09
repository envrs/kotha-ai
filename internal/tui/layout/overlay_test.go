package layout

import (
	"strings"
	"testing"
)

func TestGetLines(t *testing.T) {
	lines, widest := getLines("a\nbbb\nc")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if widest != 3 {
		t.Fatalf("expected widest 3, got %d", widest)
	}

	lines, widest = getLines("")
	if len(lines) != 1 || widest != 0 {
		t.Fatalf("expected empty single line, got %d lines, widest %d", len(lines), widest)
	}
}

func TestPlaceOverlayBasic(t *testing.T) {
	fg := "X"
	out := PlaceOverlay(0, 0, fg, "hello\nworld", false)
	if !strings.Contains(out, "X") {
		t.Fatalf("overlay should contain fg: %q", out)
	}
	if !strings.Contains(out, "world") {
		t.Fatalf("overlay should contain bg lines: %q", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("overlay should have 2 lines: %q", out)
	}
}

func TestPlaceOverlayLargerThanBg(t *testing.T) {
	fg := "very long foreground text"
	out := PlaceOverlay(0, 0, fg, "bg", false)
	if !strings.Contains(out, fg) {
		t.Fatalf("overlay should return fg when larger than bg: %q", out)
	}
}

func TestPlaceOverlayWithShadow(t *testing.T) {
	fg := "X"
	out := PlaceOverlay(0, 0, fg, "hello\nworld", true)
	if !strings.Contains(out, "X") {
		t.Fatalf("shadow overlay should contain fg: %q", out)
	}
}

func TestWhitespaceRender(t *testing.T) {
	var ws whitespace
	out := ws.render(5)
	if out != "     " {
		t.Fatalf("expected 5 spaces, got %q", out)
	}
}

func TestWhitespaceRenderWithChars(t *testing.T) {
	var ws whitespace
	ws.chars = "."
	out := ws.render(4)
	if out != "...." {
		t.Fatalf("expected 4 dots, got %q", out)
	}
}

func TestCutLeft(t *testing.T) {
	// cutLeft cuts printable characters from the left.
	if cutLeft("hello world", 5) != " world" {
		t.Fatalf("cutLeft should drop first 5 chars: %q", cutLeft("hello world", 5))
	}
}

func TestCutLeftShort(t *testing.T) {
	if out := cutLeft("hi", 10); out != "" && out != "hi" {
		t.Fatalf("unexpected cutLeft result: %q", out)
	}
}
