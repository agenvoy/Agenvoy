package knowledge

import (
	"log/slog"
	"maps"
	"slices"
	"strings"
)

const (
	DefaultLimit = 5
	MaxLimit     = 20
	trigramMin   = 3
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

func Search(keywords []string, limit int) []Hit {
	terms := normalize(keywords)
	if len(terms) == 0 || conn == nil {
		return []Hit{}
	}
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}

	counts := make(map[string]int)
	for _, term := range terms {
		for _, name := range matchNames(term) {
			counts[name]++
		}
	}
	if len(counts) == 0 {
		return []Hit{}
	}

	out := make([]Hit, 0, len(counts))
	for _, name := range slices.Sorted(maps.Keys(counts)) {
		out = append(out, Hit{Name: name, Hits: counts[name]})
	}
	slices.SortStableFunc(out, func(a, b Hit) int { return b.Hits - a.Hits })

	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func matchNames(term string) []string {
	if len([]rune(term)) >= trigramMin {
		return queryNames(`
		SELECT name
		FROM knowledge_fts5
		WHERE knowledge_fts5 MATCH ?
		`, phrase(term))
	}

	return queryNames(`
	SELECT name
	FROM knowledge
	WHERE name LIKE '%'||?||'%' OR content LIKE '%'||?||'%'
	`, term, term)
}

func phrase(term string) string {
	return `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
}

func queryNames(query string, args ...any) []string {
	rows, err := conn.Read.Query(query, args...)
	if err != nil {
		slog.Debug("knowledge match",
			slog.String("error", err.Error()))
		return nil
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil
		}
		list = append(list, name)
	}
	if rows.Err() != nil {
		return nil
	}
	return list
}
