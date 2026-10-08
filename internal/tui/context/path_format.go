package tuicontext

import (
	"os"
	"strings"
)

// ShortenHome replaces the home prefix with ~.
func ShortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// TruncateMiddle shortens s to max runes with an ellipsis in the middle.
func TruncateMiddle(s string, max int) string {
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	half := (max - 1) / 2
	return string(runes[:half]) + "…" + string(runes[len(runes)-(max-1-half):])
}

// FormatPath combines directory-relative display with home shortening.
func (d Directory) FormatPath(path string, max int) string {
	display := ShortenHome(d.Relative(path))
	return TruncateMiddle(display, max)
}
