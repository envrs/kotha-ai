package tuicontext

import (
	"os"
	"path/filepath"
)

// Project describes the detected project root.
type Project struct {
	Root        string
	Initialized bool
	Name        string
}

var projectMarkers = []string{".git", ".kotha.json", "Kotha.md", "go.mod"}

// DetectProject walks up from cwd looking for a project marker.
func DetectProject(cwd string) Project {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	dir := cwd
	for {
		for _, marker := range projectMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				_, initErr := os.Stat(filepath.Join(dir, "Kotha.md"))
				return Project{Root: dir, Initialized: initErr == nil, Name: filepath.Base(dir)}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Project{Root: cwd, Name: filepath.Base(cwd)}
		}
		dir = parent
	}
}
