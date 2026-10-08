package tuicontext

import (
	"os"
	"path/filepath"
	"strings"
)

// Directory tracks the working directory and relativizes paths for display.
type Directory struct {
	Cwd string
}

func NewDirectory(cwd string) Directory {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	return Directory{Cwd: cwd}
}

func (d Directory) Relative(path string) string {
	rel, err := filepath.Rel(d.Cwd, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}
