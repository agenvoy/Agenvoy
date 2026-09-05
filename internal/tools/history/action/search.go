package actionHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	historyStore "github.com/pardnchiu/agenvoy/internal/runtime/history"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

type historyHit struct {
	Key     string
	TS      int64
	Role    string
	Content string
}

const (
	historyScanCap      = 48
	historyWindowBefore = 2
	historyWindowAfter  = 1
	defaultTimeRange    = "1d"
	defaultSearchLimit  = 8
	maxSearchLimit      = 32
)

var historyTimeRanges = map[string]time.Duration{
	"1d": 24 * time.Hour,
	"7d": 7 * 24 * time.Hour,
	"1m": 30 * 24 * time.Hour,
	"1y": 365 * 24 * time.Hour,
}

func searchMessages(ctx context.Context, e *toolTypes.Executor, keyword, match, timeRange string, limit int) (string, error) {
	if e.SessionID == "" {
		return "", fmt.Errorf("session not exist")
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", fmt.Errorf("keyword is required when mode=search")
	}

	if match = strings.TrimSpace(match); match != "keyword" {
		match = "semantic"
	}

	if _, ok := historyTimeRanges[strings.TrimSpace(timeRange)]; !ok {
		timeRange = defaultTimeRange
	}

	if limit <= 0 {
		limit = defaultSearchLimit
	}
	limit = min(limit, maxSearchLimit)

	if e.IgnoreHistory {
		return "no history found", nil
	}

	if match == "keyword" {
		return keywordHandler(ctx, e.SessionID, keyword, timeRange, limit)
	}
	return semanticHandler(ctx, e.SessionID, keyword, timeRange, limit)
}

func keywordHandler(ctx context.Context, sessionID, keyword, timeRange string, limit int) (string, error) {
	var sb strings.Builder

	reults, err := historyStore.Search(sessionID, keyword, timeRange, limit)
	if err == nil && len(reults) > 0 {
		sb.WriteString("[archive]\n")
		for _, r := range reults {
			tsStr := time.Unix(0, r.Timestamp).Format(time.RFC3339)
			who := r.Role
			if r.Sender != "" {
				who = r.Role + " · " + r.Sender
			}
			sb.WriteString(fmt.Sprintf("[%s · %s] %s\n", tsStr, who, r.Content))
		}
	}

	db := torii.DB(torii.DBSessionHist)
	var afterNano int64
	scan := torii.ScanOption{Contains: keyword, Limit: historyScanCap}
	if d, ok := historyTimeRanges[timeRange]; ok {
		afterNano = time.Now().Add(-d).UnixNano()
		scan.After = time.Now().Add(-d).Unix()
	}

	entries := db.Scan(ctx, sessionID+":*", scan)
	if len(entries) > 0 {
		recent := keywordHits(entries, keyword, afterNano, limit)
		if len(recent) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("[recent]\n")
			for _, h := range recent {
				tsStr := time.Unix(0, h.TS).Format(time.RFC3339)
				sb.WriteString(fmt.Sprintf("[%s · %s] %s\n", tsStr, h.Role, h.Content))
			}
		}
	}

	if sb.Len() == 0 {
		return fmt.Sprintf("no matches with keyword: %s", keyword), nil
	}
	return sb.String(), nil
}

func semanticHandler(ctx context.Context, sessionID, keyword, timeRange string, limit int) (string, error) {
	db := torii.DB(torii.DBSessionHist)
	allKeys := db.Keys(ctx, sessionID+":*")
	if len(allKeys) == 0 {
		return "no history found", nil
	}

	keyIdx := make(map[string]int, len(allKeys))
	for i, one := range allKeys {
		keyIdx[one] = i
	}

	_, maxHistory := sessionHistory.Get(sessionID)
	skip := min(len(maxHistory)+1, len(allKeys))
	excludeKeys := make(map[string]struct{}, skip)
	if skip > 0 {
		for _, k := range allKeys[len(allKeys)-skip:] {
			excludeKeys[k] = struct{}{}
		}
	}

	hits := semanticHits(ctx, db, sessionID, keyword, excludeKeys, limit)
	if len(hits) == 0 {
		return fmt.Sprintf("no matches with keyword: %s", keyword), nil
	}

	expanded := expandWindows(hits, allKeys, keyIdx, excludeKeys)
	if len(expanded) == 0 {
		return fmt.Sprintf("no matches with keyword: %s", keyword), nil
	}

	return formatSegments(ctx, db, allKeys, expanded), nil
}

func keywordHits(entries []torii.Entry, keyword string, afterNano int64, cap int) []historyHit {
	lower := strings.ToLower(keyword)
	out := make([]historyHit, 0, cap)

	for _, entry := range slices.Backward(entries) {
		key := entry.Key
		ts, ok := parseKeyTS(key)
		if !ok {
			continue
		}
		if afterNano > 0 && ts < afterNano {
			break
		}

		val := entry.Value()
		if !strings.Contains(strings.ToLower(val), lower) {
			continue
		}

		hit, ok := decodeHit(key, ts, val)
		if !ok {
			continue
		}
		out = append(out, hit)
		if len(out) >= cap {
			break
		}
	}
	return out
}

func semanticHits(ctx context.Context, db *torii.Session, sessionID, keyword string, exclude map[string]struct{}, cap int) []historyHit {
	hits, err := db.VSearch(ctx, keyword, sessionID+":*", cap+len(exclude))
	if err != nil {
		return nil
	}

	out := make([]historyHit, 0, cap)
	for _, key := range hits {
		if _, skip := exclude[key]; skip {
			continue
		}
		ts, ok := parseKeyTS(key)
		if !ok {
			continue
		}
		entry, ok := db.Get(ctx, key)
		if !ok {
			continue
		}
		hit, ok := decodeHit(key, ts, entry.Value())
		if !ok {
			continue
		}
		out = append(out, hit)
		if len(out) >= cap {
			break
		}
	}
	return out
}

func expandWindows(hits []historyHit, allKeys []string, keyIdx map[string]int, exclude map[string]struct{}) []int {
	set := make(map[int]struct{}, len(hits)*(historyWindowBefore+historyWindowAfter+1))
	for _, h := range hits {
		idx, ok := keyIdx[h.Key]
		if !ok {
			continue
		}
		start := max(idx-historyWindowBefore, 0)
		end := min(idx+historyWindowAfter, len(allKeys)-1)
		for i := start; i <= end; i++ {
			if _, skip := exclude[allKeys[i]]; skip {
				continue
			}
			set[i] = struct{}{}
		}
	}

	out := make([]int, 0, len(set))
	for i := range set {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

func formatSegments(ctx context.Context, db *torii.Session, allKeys []string, idxs []int) string {
	var sb strings.Builder
	prevIdx := -1
	first := true
	for _, i := range idxs {
		if !first && i != prevIdx+1 {
			sb.WriteString("\n")
		}
		prevIdx = i

		key := allKeys[i]
		ts, ok := parseKeyTS(key)
		if !ok {
			continue
		}
		entry, ok := db.Get(ctx, key)
		if !ok {
			continue
		}
		hit, ok := decodeHit(key, ts, entry.Value())
		if !ok {
			continue
		}
		first = false
		tsStr := time.Unix(0, hit.TS).Format(time.RFC3339)
		sb.WriteString(fmt.Sprintf("[%s · %s] %s\n", tsStr, hit.Role, hit.Content))
	}
	return sb.String()
}

func parseKeyTS(key string) (int64, bool) {
	idx := strings.LastIndexByte(key, ':')
	if idx < 0 {
		return 0, false
	}
	ts, err := strconv.ParseInt(key[idx+1:], 10, 64)
	if err != nil {
		return 0, false
	}
	return ts, true
}

func decodeHit(key string, ts int64, val string) (historyHit, bool) {
	var msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		return historyHit{}, false
	}
	content := strings.TrimSpace(sessionHistory.StripPrefix(msg.Content))
	if content == "" {
		return historyHit{}, false
	}
	return historyHit{Key: key, TS: ts, Role: msg.Role, Content: content}, true
}
