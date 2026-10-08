package prompt

import (
	"sort"
	"strings"
)

// Index is a lightweight in-memory file/candidate catalog for completion.
type Index struct {
	paths map[string]struct{}
}

func (ix *Index) Add(paths ...string) {
	if ix.paths == nil {
		ix.paths = map[string]struct{}{}
	}
	for _, p := range paths {
		if p != "" {
			ix.paths[p] = struct{}{}
		}
	}
}

func (ix *Index) Remove(path string) {
	delete(ix.paths, path)
}

func (ix *Index) Len() int { return len(ix.paths) }

// Match returns paths fuzzy-matched against query (subsequence, case-insensitive),
// shortest-first for stable completion lists.
func (ix *Index) Match(query string, limit int) []string {
	q := strings.ToLower(query)
	var out []string
	for p := range ix.paths {
		if query == "" || subsequence(q, strings.ToLower(p)) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) < len(out[j])
		}
		return out[i] < out[j]
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func subsequence(q, s string) bool {
	j := 0
	for i := 0; i < len(s) && j < len(q); i++ {
		if s[i] == q[j] {
			j++
		}
	}
	return j == len(q)
}
