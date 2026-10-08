package prompt

import (
	"sort"
	"strings"
)

// Candidate is one ranked completion row.
type Candidate struct {
	Title string
	Value string
	Score float64
}

// Autocomplete ranks index matches by frecency then name.
type Autocomplete struct {
	Frecency *Frecency
}

func (a *Autocomplete) Complete(ix *Index, query string, limit int) []Candidate {
	matches := ix.Match(query, 0)
	out := make([]Candidate, 0, len(matches))
	for _, m := range matches {
		score := 0.0
		if a.Frecency != nil {
			score = a.Frecency.Score(m)
		}
		out = append(out, Candidate{Title: m, Value: m, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Title < out[j].Title
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ApplyCompletion replaces the trailing @token in text with value.
func ApplyCompletion(text, value string) string {
	prefix, _, ok := MentionToken(text)
	if !ok {
		return text
	}
	if !strings.HasSuffix(prefix, " ") && prefix != "" {
		prefix += " "
	}
	return prefix + "@" + value + " "
}
