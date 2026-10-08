package dialog

import "strings"

// FilterModels returns model names containing query (case-insensitive).
// Companion to the existing ModelDialog (models.go): reusable filtering
// logic for model pickers.
func FilterModels(models []string, query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return models
	}
	var out []string
	for _, m := range models {
		if strings.Contains(strings.ToLower(m), q) {
			out = append(out, m)
		}
	}
	return out
}
