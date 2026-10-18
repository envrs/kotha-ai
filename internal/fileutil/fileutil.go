package fileutil

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/kothagpt/kotha/internal/logging"
)

var (
	rgPath  string
	fzfPath string
)

func init() {
	var err error
	rgPath, err = exec.LookPath("rg")
	if err != nil {
		logging.Warn("Ripgrep (rg) not found in $PATH. Some features might be limited or slower.")
		rgPath = ""
	}
	fzfPath, err = exec.LookPath("fzf")
	if err != nil {
		logging.Warn("FZF not found in $PATH. Some features might be limited or slower.")
		fzfPath = ""
	}
}

func GetRgCmd(globPattern string) *exec.Cmd {
	if rgPath == "" {
		return nil
	}
	rgArgs := []string{
		"--files",
		"-L",
		"--null",
	}
	if globPattern != "" {
		if !filepath.IsAbs(globPattern) && !strings.HasPrefix(globPattern, "/") {
			globPattern = "/" + globPattern
		}
		rgArgs = append(rgArgs, "--glob", globPattern)
	}
	cmd := exec.Command(rgPath, rgArgs...)
	cmd.Dir = "."
	return cmd
}

func GetFzfCmd(query string) *exec.Cmd {
	if fzfPath == "" {
		return nil
	}
	fzfArgs := []string{
		"--filter",
		query,
		"--read0",
		"--print0",
	}
	cmd := exec.Command(fzfPath, fzfArgs...)
	cmd.Dir = "."
	return cmd
}

type FileInfo struct {
	Path    string
	ModTime time.Time
}

func SkipHidden(path string) bool {
	// Check for hidden files (starting with a dot)
	base := filepath.Base(path)
	if base != "." && strings.HasPrefix(base, ".") {
		return true
	}

	commonIgnoredDirs := map[string]bool{
		".kotha":           true,
		"node_modules":     true,
		"vendor":           true,
		"dist":             true,
		"build":            true,
		"target":           true,
		".git":             true,
		".idea":            true,
		".vscode":          true,
		"__pycache__":      true,
		"bin":              true,
		"obj":              true,
		"out":              true,
		"coverage":         true,
		"tmp":              true,
		"temp":             true,
		"logs":             true,
		"generated":        true,
		"bower_components": true,
		"jspm_packages":    true,
	}

	parts := strings.Split(path, string(os.PathSeparator))
	for _, part := range parts {
		if commonIgnoredDirs[part] {
			return true
		}
	}
	return false
}

func GlobWithDoublestar(pattern, searchPath string, limit int) ([]string, bool, error) {
	fsys := os.DirFS(searchPath)
	relPattern := strings.TrimPrefix(pattern, "/")
	var matches []FileInfo

	err := doublestar.GlobWalk(fsys, relPattern, func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil
		}
		if SkipHidden(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		absPath := path
		if !strings.HasPrefix(absPath, searchPath) && searchPath != "." {
			absPath = filepath.Join(searchPath, absPath)
		} else if !strings.HasPrefix(absPath, "/") && searchPath == "." {
			absPath = filepath.Join(searchPath, absPath) // Ensure relative paths are joined correctly
		}

		matches = append(matches, FileInfo{Path: absPath, ModTime: info.ModTime()})
		if limit > 0 && len(matches) >= limit*2 {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("glob walk error: %w", err)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	truncated := false
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
		truncated = true
	}

	results := make([]string, len(matches))
	for i, m := range matches {
		results[i] = m.Path
	}
	return results, truncated, nil
}

// ExpandTilde expands a leading tilde (~) in a path to the user's home directory.
// It handles:
//
//	- ~         -> $HOME
//	- ~/path    -> $HOME/path
//	- ~\path    -> $HOME\path (Windows)
//	- ~user/... -> (not expanded, left as-is for security)
//
// Paths like ~draft.md (no separator after ~) are left unchanged.
// Returns the original path if no expansion is needed or possible.
func ExpandTilde(path string) string {
	if path == "" {
		return path
	}

	// Check for leading tilde
	if !strings.HasPrefix(path, "~") {
		return path
	}

	// If path is just "~" or starts with "~/" or "~\"
	if len(path) == 1 || (len(path) > 1 && (path[1] == '/' || (runtime.GOOS == "windows" && path[1] == '\\'))) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			// Fallback: try environment variables
			home = os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home == "" {
				return path // Cannot expand, return original
			}
		}
		if len(path) == 1 {
			return home
		}
		return filepath.Join(home, path[2:])
	}

	// Path starts with ~ but not followed by separator (e.g., ~draft.md)
	// Do not expand - leave as literal
	return path
}

// MustExpandTilde is like ExpandTilde but panics if expansion fails.
// Use only when the path is guaranteed to be expandable (e.g., in tests).
func MustExpandTilde(path string) string {
	if path == "" {
		panic("expand tilde: empty path")
	}
	expanded := ExpandTilde(path)
	if expanded == path && strings.HasPrefix(path, "~") {
		// Path starts with ~ but wasn't expanded (e.g., ~user/file)
		// This is a security feature - we don't expand ~user/ paths
		// But for MustExpandTilde, we consider this a failure to expand
		panic(fmt.Sprintf("expand tilde: cannot expand %q", path))
	}
	return expanded
}
