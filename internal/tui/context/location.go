package tuicontext

import (
	"fmt"
	"strconv"
	"strings"
)

// Location is a file position for jump-to / diagnostics display.
type Location struct {
	Path string
	Line int
	Col  int
}

func (l Location) String() string {
	if l.Line <= 0 {
		return l.Path
	}
	if l.Col <= 0 {
		return fmt.Sprintf("%s:%d", l.Path, l.Line)
	}
	return fmt.Sprintf("%s:%d:%d", l.Path, l.Line, l.Col)
}

// ParseLocation parses "path[:line[:col]]".
func ParseLocation(s string) Location {
	parts := strings.Split(s, ":")
	loc := Location{Path: parts[0]}
	if len(parts) > 1 {
		if n, err := strconv.Atoi(parts[1]); err == nil {
			loc.Line = n
		}
	}
	if len(parts) > 2 {
		if n, err := strconv.Atoi(parts[2]); err == nil {
			loc.Col = n
		}
	}
	return loc
}
