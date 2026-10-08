package prompt

import "time"

// Frecency scores candidates by access frequency decayed by recency.
// Score = hits / (1 + age_hours). Cheap, deterministic, testable.
type Frecency struct {
	hits map[string]int
	last map[string]time.Time
	now  func() time.Time
}

func NewFrecency() *Frecency {
	return &Frecency{hits: map[string]int{}, last: map[string]time.Time{}, now: time.Now}
}

func (f *Frecency) Record(key string) {
	if f.hits == nil {
		f.hits = map[string]int{}
		f.last = map[string]time.Time{}
	}
	if f.now == nil {
		f.now = time.Now
	}
	f.hits[key]++
	f.last[key] = f.now()
}

func (f *Frecency) Score(key string) float64 {
	hits := f.hits[key]
	if hits == 0 {
		return 0
	}
	age := 0.0
	if t, ok := f.last[key]; ok && f.now != nil {
		age = f.now().Sub(t).Hours()
		if age < 0 {
			age = 0
		}
	}
	return float64(hits) / (1 + age)
}
