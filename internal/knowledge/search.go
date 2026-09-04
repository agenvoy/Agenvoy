package knowledge

import (
	"sort"
	"strings"
)

const (
	DefaultLimit = 5
	MaxLimit     = 20
)

type Hit struct {
	Name string `json:"name"`
	Hits int    `json:"hits"`
}

func normalize(keywords []string) []string {
	terms := make([]string, 0, len(keywords))
	for _, one := range keywords {
		if one = strings.ToLower(strings.TrimSpace(one)); one != "" {
			terms = append(terms, one)
		}
	}
	return terms
}

func ListNames(keywords []string) []string {
	terms := normalize(keywords)
	if len(terms) == 0 {
		return nil
	}

	out := []string{}
	for _, record := range List() {
		name := strings.ToLower(record.Name)
		for _, term := range terms {
			if strings.Contains(name, term) {
				out = append(out, record.Name)
				break
			}
		}
	}
	return out
}

func Search(keywords []string, limit int) []Hit {
	terms := normalize(keywords)
	if len(terms) == 0 {
		return []Hit{}
	}
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}

	out := make([]Hit, 0, limit)
	for _, record := range List() {
		body := strings.ToLower(record.Name + "\n" + record.Content)
		matched := 0
		for _, term := range terms {
			if strings.Contains(body, term) {
				matched++
			}
		}
		if matched == 0 {
			continue
		}
		out = append(out, Hit{Name: record.Name, Hits: matched})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Hits > out[j].Hits })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
