package fileutil

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// GitIgnore evaluates .gitignore-format exclusion patterns collected from
// one or more .gitignore files under a root directory. Nested .gitignore
// files override shallower ones, and within a file later lines win —
// matching git's own precedence rules.
//
// Supported syntax: comments (#), negation (!), directory-only (/),
// anchoring (interior or leading /), and ** globs. Brace expansion {a,b}
// is not part of gitignore but doublestar matches it anyway.
type GitIgnore struct {
	root  string
	rules []giRule
}

type giRule struct {
	base     string // slash-path of the .gitignore's dir relative to root ("" = root)
	depth    int    // base depth, used for precedence
	negate   bool
	dirOnly  bool
	anchored bool
	pattern  string
}

// NewGitIgnore creates a matcher rooted at root. Call LoadDir to read
// .gitignore files.
func NewGitIgnore(root string) *GitIgnore {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = filepath.Clean(root)
	}
	return &GitIgnore{root: abs}
}

// LoadDir reads dir/.gitignore (relative to the matcher root) and adds
// its rules. Missing files are not an error. Directories outside root are
// ignored.
func (g *GitIgnore) LoadDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(g.root, abs)
	if err != nil {
		return nil
	}
	if rel == "." {
		rel = ""
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil
	}
	rel = filepath.ToSlash(rel)

	data, err := os.ReadFile(filepath.Join(abs, ".gitignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	g.AddRules(rel, string(data))
	return nil
}

// AddRules parses gitignore text whose patterns are relative to base
// (slash-separated path from root; "" means root).
func (g *GitIgnore) AddRules(base, content string) {
	depth := 0
	if base != "" {
		depth = strings.Count(base, "/") + 1
	}
	for _, line := range strings.Split(content, "\n") {
		if r, ok := parseGILine(line); ok {
			r.base = base
			r.depth = depth
			g.rules = append(g.rules, r)
		}
	}
	sort.SliceStable(g.rules, func(a, b int) bool {
		return g.rules[a].depth < g.rules[b].depth
	})
}

// Match reports whether relPath (slash-separated, relative to root) is
// ignored. An ignored ancestor directory ignores everything beneath it,
// even when a deeper rule negates the file itself — same as git.
func (g *GitIgnore) Match(relPath string, isDir bool) bool {
	relPath = strings.TrimPrefix(path.Clean(relPath), "./")
	if relPath == "." || relPath == "" || relPath == "/" {
		return false
	}
	parts := strings.Split(relPath, "/")
	for i := 1; i < len(parts); i++ {
		if g.matchOne(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return g.matchOne(relPath, isDir)
}

// MatchAbs reports whether an absolute path is ignored. Paths outside the
// matcher root are never ignored.
func (g *GitIgnore) MatchAbs(absPath string, isDir bool) bool {
	rel, err := filepath.Rel(g.root, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return g.Match(filepath.ToSlash(rel), isDir)
}

// Root returns the matcher root directory.
func (g *GitIgnore) Root() string { return g.root }

// matchOne evaluates all rules against a single path; the last matching
// rule decides.
func (g *GitIgnore) matchOne(rel string, isDir bool) bool {
	ignored := false
	for _, r := range g.rules {
		sub, ok := r.relTo(rel)
		if !ok {
			continue
		}
		if r.dirOnly && !isDir {
			continue
		}
		var matched bool
		if r.anchored {
			matched, _ = doublestar.Match(r.pattern, sub)
		} else {
			// Patterns without an interior slash match at any depth.
			matched, _ = doublestar.Match(r.pattern, path.Base(sub))
		}
		if matched {
			ignored = !r.negate
		}
	}
	return ignored
}

// relTo strips the rule's base from rel, reporting whether rel lives
// under that base.
func (r giRule) relTo(rel string) (string, bool) {
	if r.base == "" {
		return rel, true
	}
	if rel == r.base || !strings.HasPrefix(rel, r.base+"/") {
		return "", false
	}
	return rel[len(r.base)+1:], true
}

// parseGILine parses one .gitignore line.
func parseGILine(raw string) (giRule, bool) {
	line := trimGIComment(raw)
	if line == "" {
		return giRule{}, false
	}
	var r giRule
	if strings.HasPrefix(line, `\#`) || strings.HasPrefix(line, `\!`) {
		line = line[1:]
	} else {
		if strings.HasPrefix(line, "!") {
			r.negate = true
			line = line[1:]
		}
		if strings.HasPrefix(line, "#") {
			return giRule{}, false
		}
	}
	r.dirOnly = strings.HasSuffix(line, "/")
	line = strings.TrimSuffix(line, "/")
	// A slash anywhere (besides trailing) anchors the pattern to the
	// .gitignore's own directory; a leading slash is stripped.
	r.anchored = strings.Contains(line, "/")
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		return giRule{}, false
	}
	r.pattern = line
	return r, true
}

// trimGIComment strips comments and unescaped trailing whitespace.
func trimGIComment(line string) string {
	if strings.HasPrefix(line, "#") {
		return ""
	}
	if strings.HasSuffix(line, `\ `) {
		// Trailing space quoted with a backslash survives as a literal.
		return strings.TrimRight(line[:len(line)-2], " \t\r") + " "
	}
	return strings.TrimRightFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r'
	})
}
